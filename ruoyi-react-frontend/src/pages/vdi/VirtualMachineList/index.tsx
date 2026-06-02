import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  Table,
  Button,
  Space,
  Tag,
  Modal,
  message,
  Popconfirm,
  Select,
  Input,
  Card,
  Tooltip,
  Form,
  Slider,
  InputNumber,
  Row,
  Col,
  Spin,
} from 'antd';
import {
  ReloadOutlined,
  PlayCircleOutlined,
  StopOutlined,
  SyncOutlined,
  DeleteOutlined,
  PlusOutlined,
  EditOutlined,
  SettingOutlined,
  UserAddOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { vmApi, vdiServerApi } from '@/lib/vdiApi';
import type { VirtualMachine, VMListParams, CreateVMRequest, VDIServer, VDIResourceGroup, VDIResource, VDIPlatform, RunPosition, VDIStorage, VDINetwork } from '@/types/vdi';
import type { ColumnsType } from 'antd/es/table';

const VirtualMachineList: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [vms, setVMs] = useState<VirtualMachine[]>([]);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [filters, setFilters] = useState<Partial<VMListParams>>({});
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [configIPModalVisible, setConfigIPModalVisible] = useState(false);
  const [renameModalVisible, setRenameModalVisible] = useState(false);
  const [bindUserModalVisible, setBindUserModalVisible] = useState(false);
  const [selectedVM, setSelectedVM] = useState<VirtualMachine | null>(null);
  const [form] = Form.useForm();
  const [vdiServers, setVdiServers] = useState<VDIServer[]>([]);
  const [resourceGroups, setResourceGroups] = useState<VDIResourceGroup[]>([]);
  const [resources, setResources] = useState<VDIResource[]>([]);
  const [vtpPlatforms, setVtpPlatforms] = useState<VDIPlatform[]>([]);
  const [runPositions, setRunPositions] = useState<RunPosition[]>([]);
  const [storages, setStorages] = useState<VDIStorage[]>([]);
  const [networks, setNetworks] = useState<VDINetwork[]>([]);

  // VDI数据预加载状态
  const [vdiDataPreloaded, setVdiDataPreloaded] = useState(false);
  const [vdiDataLoading, setVdiDataLoading] = useState(false);

  // API缓存：5分钟过期
  const vdiDataCache = useRef<{
    vtpPlatforms: VDIPlatform[];
    runPositions: RunPosition[];
    storages: VDIStorage[];
    networks: VDINetwork[];
    timestamp: number;
  } | null>(null);
  const CACHE_DURATION = 5 * 60 * 1000; // 5分钟

  // 预加载VDI配置数据（页面加载时就开始）
  const preloadVDIData = useCallback(async () => {
    // 检查缓存
    if (vdiDataCache.current && (Date.now() - vdiDataCache.current.timestamp < CACHE_DURATION)) {
      console.log('[VDI Preload] 使用缓存数据，跳过API调用');
      setVdiDataPreloaded(true);
      return;
    }

    console.log('[VDI Preload] 开始后台预加载VDI配置数据...');
    setVdiDataLoading(true);

    try {
      // 获取第一个可用的VDI服务器
      const serverResult = await vdiServerApi.list({ current: 1, pageSize: 100 });
      const servers = serverResult.data?.list || [];
      const availableServer = servers.find(s => s.status === 0);

      if (!availableServer) {
        console.log('[VDI Preload] 没有可用的VDI服务器');
        setVdiDataLoading(false);
        return;
      }

      console.log('[VDI Preload] 使用VDI服务器:', availableServer.name);

      // 并行加载所有VDI配置数据
      const [platformsResult, positionsResult, storagesResult, networksResult] = await Promise.all([
        vmApi.listVTPPlatforms(availableServer.id),
        vmApi.listRunPositions(availableServer.id, 1), // 假设VTP ID为1
        vmApi.listStorages(availableServer.id, 1),
        vmApi.listNetworks(availableServer.id, 1),
      ]);

      const newPlatforms = platformsResult.data || [];
      const newPositions = positionsResult.data || [];
      const newStorages = storagesResult.data || [];
      const newNetworks = networksResult.data || [];

      // 更新缓存
      vdiDataCache.current = {
        vtpPlatforms: newPlatforms,
        runPositions: newPositions,
        storages: newStorages,
        networks: newNetworks,
        timestamp: Date.now(),
      };

      setVtpPlatforms(newPlatforms);
      setRunPositions(newPositions);
      setStorages(newStorages);
      setNetworks(newNetworks);
      setVdiDataPreloaded(true);

      console.log('[VDI Preload] 预加载完成:', {
        platforms: newPlatforms.length,
        positions: newPositions.length,
        storages: newStorages.length,
        networks: newNetworks.length,
      });
    } catch (error) {
      console.error('[VDI Preload] 预加载失败:', error);
    } finally {
      setVdiDataLoading(false);
    }
  }, []);

  // 加载虚拟机列表
  const loadVMs = async () => {
    setLoading(true);
    try {
      const params: VMListParams = {
        current,
        pageSize,
        ...filters,
      };
      const result = await vmApi.list(params);
      setVMs(result.data?.list || []);
      setTotal(result.data?.total || 0);
    } catch (error) {
      message.error('加载虚拟机列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadVMs();
  }, [current, pageSize, filters]);

  // 页面加载时预加载VDI配置数据（不阻塞UI渲染）
  useEffect(() => {
    // 使用 setTimeout 让出主线程，确保页面先渲染
    const timer = setTimeout(() => {
      preloadVDIData();
    }, 100);
    return () => clearTimeout(timer);
  }, [preloadVDIData]);

  // Watch form field changes for cascading dropdowns and auto-name
  const selectedServerId = Form.useWatch('vdi_server_id', form);
  const selectedResourceGroupId = Form.useWatch('resource_group_id', form);
  const selectedResourceFieldId = Form.useWatch('resource_id', form);
  const selectedVtpId = Form.useWatch('vtp_id', form);

  // Load resource groups when VDI server is selected
  useEffect(() => {
    if (selectedServerId && createModalVisible) {
      vmApi.listResourceGroups(selectedServerId).then(result => {
        setResourceGroups(result.data || []);
      }).catch(() => {
        setResourceGroups([]);
      });
      // Clear resources when VDI server changes
      setResources([]);
      form.setFieldsValue({ resource_group_id: undefined, resource_id: undefined });
    } else {
      setResourceGroups([]);
    }
  }, [selectedServerId, createModalVisible, form]);

  // Load resources when resource group is selected
  useEffect(() => {
    if (selectedResourceGroupId && selectedServerId && createModalVisible) {
      vmApi.listResources(selectedServerId, selectedResourceGroupId).then(result => {
        const list = result.data || [];
        setResources(list);
        // Auto-select first resource
        if (list.length > 0) {
          form.setFieldsValue({ resource_id: String(list[0].id) });
        }
      }).catch(() => {
        setResources([]);
      });
    } else {
      setResources([]);
      form.setFieldsValue({ resource_id: undefined });
    }
  }, [selectedResourceGroupId, selectedServerId, createModalVisible, form]);

  // Auto-generate name when resource and VTP are selected (优化命名格式)
  useEffect(() => {
    if (selectedResourceGroupId && selectedResourceFieldId) {
      const group = resourceGroups.find(g => g.resource_group_id === selectedResourceGroupId);
      const resource = resources.find(r => String(r.id) === selectedResourceFieldId);
      const suffix = form.getFieldValue('name_suffix') || '';

      if (group && resource) {
        // 优化命名: VDI-{资源组}-{资源}-{后缀/序号}
        // 示例: VDI-研发部-资源池1-zhangsan-001
        const baseName = `VDI-${group.name}-${resource.name}`;
        form.setFieldsValue({
          name: suffix ? `${baseName}-${suffix}` : baseName
        });
      }
    }
  }, [selectedResourceGroupId, selectedResourceFieldId, resourceGroups, resources, form]);

  // Load VTP platforms when VDI server is selected (带缓存优化)
  useEffect(() => {
    if (selectedServerId && createModalVisible) {
      // 检查缓存
      const now = Date.now();
      if (vdiDataCache.current && (now - vdiDataCache.current.timestamp < CACHE_DURATION)) {
        // 使用缓存数据
        setVtpPlatforms(vdiDataCache.current.vtpPlatforms);
        setRunPositions(vdiDataCache.current.runPositions);
        setStorages(vdiDataCache.current.storages);
        setNetworks(vdiDataCache.current.networks);
        return;
      }

      vmApi.listVTPPlatforms(selectedServerId).then(result => {
        setVtpPlatforms(result.data || []);
      }).catch(() => {
        setVtpPlatforms([]);
      });
      // Clear dependent fields when VDI server changes
      setRunPositions([]);
      setStorages([]);
      setNetworks([]);
      form.setFieldsValue({
        vtp_id: undefined,
        host_id: undefined,
        run_position_id: undefined,
        disk_id: undefined,
        storage_id: undefined,
        network_id: undefined,
      });
    } else {
      setVtpPlatforms([]);
    }
  }, [selectedServerId, createModalVisible, form]);

  // Load run positions, storages, and networks when VTP platform is selected (带缓存更新)
  useEffect(() => {
    if (selectedVtpId && selectedServerId && createModalVisible) {
      let positionsLoaded = false;
      let storagesLoaded = false;
      let networksLoaded = false;

      // 检查是否需要从API加载
      const useCache = vdiDataCache.current &&
        (Date.now() - vdiDataCache.current.timestamp < CACHE_DURATION);

      // Load run positions
      const loadPositions = () => {
        if (useCache && vdiDataCache.current!.runPositions.length > 0) {
          setRunPositions(vdiDataCache.current!.runPositions);
          positionsLoaded = true;
        } else {
          return vmApi.listRunPositions(selectedServerId, selectedVtpId);
        }
      };

      // Load storages
      const loadStorages = () => {
        if (useCache && vdiDataCache.current!.storages.length > 0) {
          setStorages(vdiDataCache.current!.storages);
          storagesLoaded = true;
        } else {
          return vmApi.listStorages(selectedServerId, selectedVtpId);
        }
      };

      // Load networks
      const loadNetworks = () => {
        if (useCache && vdiDataCache.current!.networks.length > 0) {
          setNetworks(vdiDataCache.current!.networks);
          networksLoaded = true;
        } else {
          return vmApi.listNetworks(selectedServerId, selectedVtpId);
        }
      };

      // 如果使用缓存且所有数据都存在，直接返回
      if (useCache &&
          vdiDataCache.current!.runPositions.length > 0 &&
          vdiDataCache.current!.storages.length > 0 &&
          vdiDataCache.current!.networks.length > 0) {
        // 从缓存恢复选中状态
        const cached = vdiDataCache.current!;
        setRunPositions(cached.runPositions);
        setStorages(cached.storages);
        setNetworks(cached.networks);

        // 恢复自动选择逻辑
        if (cached.runPositions.length > 0) {
          const firstPosition = cached.runPositions[0];
          form.setFieldsValue({
            host_id: firstPosition.father_id,
            run_position_id: firstPosition.id !== firstPosition.father_id ? firstPosition.id : undefined,
          });
        }
        if (cached.storages.length > 0) {
          form.setFieldsValue({ disk_id: cached.storages[0].id, storage_id: cached.storages[0].id });
        }
        if (cached.networks.length > 0) {
          form.setFieldsValue({ network_id: cached.networks[0].id });
        }
        return;
      }

      // 从API加载数据
      Promise.all([
        positionsLoaded ? Promise.resolve({ data: [] }) : vmApi.listRunPositions(selectedServerId, selectedVtpId),
        storagesLoaded ? Promise.resolve({ data: [] }) : vmApi.listStorages(selectedServerId, selectedVtpId),
        networksLoaded ? Promise.resolve({ data: [] }) : vmApi.listNetworks(selectedServerId, selectedVtpId),
      ]).then(([positionsResult, storagesResult, networksResult]) => {
        let newPositions = positionsResult.data || [];
        const newStorages = storagesResult.data || [];
        const newNetworks = networksResult.data || [];

        // 过滤：只显示clustered virtual machine下的运行位置
        // 查找father_id包含"clustered"的位置，找到对应的ID
        const clusteredHostId = newPositions.find(p =>
          p.father_id && p.father_id.toLowerCase().includes('clustered')
        )?.father_id;

        if (clusteredHostId) {
          // 只保留该host下的运行位置
          newPositions = newPositions.filter(p => p.father_id === clusteredHostId);
          console.log('[VDI Positions] 已过滤，只显示clustered下的位置:', {
            clusteredHostId,
            count: newPositions.length,
            total: positionsResult.data?.length || 0
          });
        } else {
          console.warn('[VDI Positions] 未找到clustered host，显示所有位置');
        }

        setRunPositions(newPositions);
        setStorages(newStorages);
        setNetworks(newNetworks);

        // 更新缓存
        vdiDataCache.current = {
          vtpPlatforms,
          runPositions: newPositions,
          storages: newStorages,
          networks: newNetworks,
          timestamp: Date.now(),
        };

        // Auto-select first options if available
        if (newPositions.length > 0) {
          const firstPosition = newPositions[0];
          form.setFieldsValue({
            host_id: firstPosition.father_id,
            run_position_id: firstPosition.id !== firstPosition.father_id ? firstPosition.id : undefined,
          });
        }
        if (newStorages.length > 0) {
          form.setFieldsValue({ disk_id: newStorages[0].id, storage_id: newStorages[0].id });
        }
        if (newNetworks.length > 0) {
          form.setFieldsValue({ network_id: newNetworks[0].id });
        }
      }).catch(() => {
        setRunPositions([]);
        setStorages([]);
        setNetworks([]);
      });
    } else {
      setRunPositions([]);
      setStorages([]);
      setNetworks([]);
      form.setFieldsValue({
        host_id: undefined,
        run_position_id: undefined,
        disk_id: undefined,
        storage_id: undefined,
        network_id: undefined,
      });
    }
  }, [selectedVtpId, selectedServerId, createModalVisible, form, vtpPlatforms]);

  // Open create modal and load dropdown data (使用预加载数据)
  const openCreateModal = async () => {
    setCreateModalVisible(true);
    form.resetFields();

    // 如果已有预加载数据，直接使用
    if (vdiDataPreloaded && vdiDataCache.current) {
      console.log('[VDI Modal] 使用预加载数据，无需等待API调用');
      setVtpPlatforms(vdiDataCache.current.vtpPlatforms);
      setRunPositions(vdiDataCache.current.runPositions);
      setStorages(vdiDataCache.current.storages);
      setNetworks(vdiDataCache.current.networks);

      // 加载VDI服务器列表（这个还是需要调用API）
      try {
        const serverResult = await vdiServerApi.list({ current: 1, pageSize: 100 });
        setVdiServers(serverResult.data?.list || []);
      } catch (e) {
        // ignore
      }
      return;
    }

    // 如果没有预加载数据，显示加载状态并开始加载
    console.log('[VDI Modal] 没有预加载数据，开始加载...');
    setVtpPlatforms([]);
    setRunPositions([]);
    setStorages([]);
    setNetworks([]);
    setResources([]);

    try {
      const serverResult = await vdiServerApi.list({ current: 1, pageSize: 100 });
      setVdiServers(serverResult.data?.list || []);

      // 触发VDI数据加载
      const availableServer = serverResult.data?.list.find(s => s.status === 0);
      if (availableServer) {
        // 这会触发useEffect加载VDI数据
        form.setFieldsValue({ vdi_server_id: availableServer.id });
      }
    } catch (e) {
      // ignore
    }
  };

  // 创建虚拟机（调用 VDI API）
  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      // Implement host/run position logic from vdi_test_standalone.go
      // Extract host_id (father_id) and run_position_id (id) from selected position
      let hostId = values.host_id;
      let runPositionId = values.run_position_id;

      // Find the selected position to extract father_id logic
      if (values.run_position_id) {
        const selectedPosition = runPositions.find(p => p.id === values.run_position_id);
        if (selectedPosition) {
          hostId = selectedPosition.father_id;
          // If id == father_id, run_position_id should be empty
          if (selectedPosition.id === selectedPosition.father_id) {
            runPositionId = undefined;
          }
        }
      }

      const data: CreateVMRequest = {
        ...values,
        host_id: hostId,
        run_position_id: runPositionId,
      };

      await vmApi.create(data);
      message.success('虚拟机创建成功，VDI API 调用完成');
      setCreateModalVisible(false);
      form.resetFields();
      loadVMs();
    } catch (error) {
      message.error('创建虚拟机失败');
    } finally {
      setLoading(false);
    }
  };

  // 操作虚拟机（调用 VDI API）
  const handleOperate = async (action: string, vmIds?: string[]) => {
    const ids = vmIds || selectedRowKeys as string[];
    if (ids.length === 0) {
      message.warning('请选择要操作的虚拟机');
      return;
    }

    setLoading(true);
    try {
      await vmApi.batchOperate(ids, action);
      message.success(`${action === 'start' ? '开机' : action === 'stop' ? '关机' : '重启'}操作已提交，VDI API 调用成功`);
      setSelectedRowKeys([]);
      loadVMs();
    } catch (error) {
      message.error('操作失败，VDI API 调用失败');
    } finally {
      setLoading(false);
    }
  };

  // 删除虚拟机（调用 VDI API）
  const handleDelete = async (id: string) => {
    setLoading(true);
    try {
      await vmApi.delete(id);
      message.success('删除成功，VDI API 调用完成');
      loadVMs();
    } catch (error) {
      message.error('删除失败，VDI API 调用失败');
    } finally {
      setLoading(false);
    }
  };

  // 同步虚拟机（调用 VDI API）
  const handleSync = async (id: string) => {
    setLoading(true);
    try {
      await vmApi.sync(id);
      message.success('同步成功，VDI 状态已更新');
      loadVMs();
    } catch (error) {
      message.error('同步失败');
    } finally {
      setLoading(false);
    }
  };

  // 配置 IP（调用 VDI API）
  const handleConfigIP = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      await vmApi.configIP([{
        vm_id: selectedVM!.vm_id,
        ip_address: values.ip_address,
        netmask: values.netmask,
        gateway: values.gateway,
      }]);
      message.success('IP 配置成功，VDI API 调用完成');
      setConfigIPModalVisible(false);
      form.resetFields();
      loadVMs();
    } catch (error) {
      message.error('配置 IP 失败，VDI API 调用失败');
    } finally {
      setLoading(false);
    }
  };

  // 重命名（调用 VDI API）
  const handleRename = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      await vmApi.rename(selectedVM!.id, { new_name: values.new_name });
      message.success('重命名成功，VDI API 调用完成');
      setRenameModalVisible(false);
      form.resetFields();
      loadVMs();
    } catch (error) {
      message.error('重命名失败，VDI API 调用失败');
    } finally {
      setLoading(false);
    }
  };

  // 绑定用户（调用 VDI API）
  const handleBindUser = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      await vmApi.bindUser(selectedVM!.id, { user_id: values.user_id });
      message.success('用户绑定成功，VDI API 调用完成');
      setBindUserModalVisible(false);
      form.resetFields();
      loadVMs();
    } catch (error) {
      message.error('绑定用户失败，VDI API 调用失败');
    } finally {
      setLoading(false);
    }
  };

  // 表格列定义（包含所有操作按钮）
  const columns: ColumnsType<VirtualMachine> = [
    {
      title: '虚拟机 ID',
      dataIndex: 'vm_id',
      key: 'vm_id',
      width: 120,
      ellipsis: true,
    },
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 150,
      render: (text: string, record) => (
        <a
          onClick={() => navigate(`/vdi/vm/${record.id}`)}
          style={{ color: '#1890ff', textDecoration: 'underline' }}
        >
          {text}
        </a>
      ),
    },
    {
      title: '虚拟机状态',
      dataIndex: 'power_state',
      key: 'power_state',
      width: 100,
      render: (state: string) => {
        const config = {
          pending: { color: 'blue', text: '等待Agent' },
          stopped: { color: 'red', text: '关机' },
          suspended: { color: 'orange', text: '挂起' },
          in_use: { color: 'lime', text: '正常使用' },
        };
        const { color, text } = config[state as keyof typeof config] || { color: 'default', text: state };
        return <Tag color={color}>{text}</Tag>;
      },
    },
    {
      title: 'IP 地址',
      dataIndex: 'ip_address',
      key: 'ip_address',
      width: 120,
      render: (ip: string, record: any) => {
        const ipType = record.ip_type || 'DHCP';
        return ip ? (
          <Tooltip title={`类型: ${ipType === 'STATIC' ? '静态IP' : 'DHCP'}${record.subnet_mask ? `\n子网掩码: ${record.subnet_mask}` : ''}${record.default_gateway ? `\n网关: ${record.default_gateway}` : ''}${record.name_server ? `\nDNS: ${record.name_server}` : ''}`}>
            <span>{ip} ({ipType === 'STATIC' ? '静态' : 'DHCP'})</span>
          </Tooltip>
        ) : '-';
      },
    },
    {
      title: '操作系统',
      dataIndex: 'os_type',
      key: 'os_type',
      width: 120,
    },
    {
      title: 'CPU/内存/磁盘',
      key: 'resources',
      width: 200,
      render: (_, record) => {
        const cpuNumber = record.cpu_number || 0;
        const cpuCore = record.cpu_core || 0;
        const cpuPer = record.cpu_per || 0;
        const memoryMB = parseInt(String(record.memory || '0'), 10);
        const memoryPer = record.memory_per || 0;
        const diskMB = parseInt(String(record.disk || '0'), 10);
        const diskPer = record.disk_per || 0;

        // 转换为 GB
        const memoryGB = (memoryMB / 1024).toFixed(1);
        const diskGB = (diskMB / 1024).toFixed(1);

        // 构建CPU显示字符串
        let cpuDisplay = '';
        if (cpuNumber > 0 && cpuCore > 0) {
          cpuDisplay = `${cpuNumber}颗 × ${cpuCore}核`;
          if (cpuPer > 0) {
            cpuDisplay += ` (${cpuPer}%)`;
          }
        } else if (cpuNumber > 0) {
          cpuDisplay = `${cpuNumber}核`;
        } else {
          cpuDisplay = '-';
        }

        // 构建内存显示字符串
        let memoryDisplay = `${memoryGB}GB`;
        if (memoryPer > 0) {
          memoryDisplay += ` (${memoryPer}%)`;
        }

        // 构建磁盘显示字符串
        let diskDisplay = `${diskGB}GB`;
        if (diskPer > 0) {
          diskDisplay += ` (${diskPer}%)`;
        }

        return `${cpuDisplay} / ${memoryDisplay} / ${diskDisplay}`;
      },
    },
    {
      title: '绑定用户',
      dataIndex: 'bound_user_name',
      key: 'bound_user_name',
      width: 120,
      render: (name: string) => name || '-',
    },
    {
      title: '最后同步',
      dataIndex: 'last_sync_at',
      key: 'last_sync_at',
      width: 160,
      render: (time: string) => time ? new Date(time).toLocaleString('zh-CN') : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 300,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="开机">
            <Button
              type="text"
              icon={<PlayCircleOutlined />}
              onClick={() => handleOperate('start', [record.id])}
              disabled={record.power_state !== 'stopped'}
            />
          </Tooltip>
          <Tooltip title="关机">
            <Button
              type="text"
              icon={<StopOutlined />}
              onClick={() => handleOperate('stop', [record.id])}
              disabled={record.power_state === 'stopped' || record.power_state === 'suspended' || record.power_state === 'pending'}
            />
          </Tooltip>
          <Tooltip title="重启">
            <Button
              type="text"
              icon={<SyncOutlined />}
              onClick={() => handleOperate('restart', [record.id])}
              disabled={record.power_state !== 'in_use' && record.power_state !== 'suspended'}
            />
          </Tooltip>
          <Tooltip title="同步">
            <Button
              type="text"
              icon={<SyncOutlined />}
              onClick={() => handleSync(record.id)}
            />
          </Tooltip>
          <Tooltip title="配置 IP">
            <Button
              type="text"
              icon={<SettingOutlined />}
              onClick={() => {
                setSelectedVM(record);
                setConfigIPModalVisible(true);
              }}
            />
          </Tooltip>
          <Tooltip title="重命名">
            <Button
              type="text"
              icon={<EditOutlined />}
              onClick={() => {
                setSelectedVM(record);
                setRenameModalVisible(true);
              }}
            />
          </Tooltip>
          <Tooltip title="绑定用户">
            <Button
              type="text"
              icon={<UserAddOutlined />}
              onClick={() => {
                setSelectedVM(record);
                setBindUserModalVisible(true);
              }}
            />
          </Tooltip>
          <Popconfirm
            title="确定要删除这个虚拟机吗？此操作将调用 VDI API 删除虚拟机。"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Card>
      <Space orientation="vertical" size="large" style={{ width: '100%' }}>
        {/* 工具栏 */}
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
            创建虚拟机
          </Button>
          <Input.Search
            placeholder="搜索虚拟机名称"
            style={{ width: 200 }}
            onSearch={(value) => setFilters({ ...filters, name: value })}
          />
          <Select
            placeholder="虚拟机状态"
            style={{ width: 120 }}
            allowClear
            onChange={(value) => setFilters({ ...filters, powerState: value })}
          >
            <Select.Option value="pending">等待Agent</Select.Option>
            <Select.Option value="stopped">关机</Select.Option>
            <Select.Option value="suspended">挂起</Select.Option>
            <Select.Option value="in_use">正常使用</Select.Option>
          </Select>
          <Button icon={<ReloadOutlined />} onClick={loadVMs}>
            刷新
          </Button>
        </Space>

        {/* 批量操作 */}
        {selectedRowKeys.length > 0 && (
          <Space>
            <span>已选择 {selectedRowKeys.length} 项</span>
            <Button onClick={() => handleOperate('start')}>批量开机</Button>
            <Button onClick={() => handleOperate('stop')}>批量关机</Button>
            <Button onClick={() => handleOperate('restart')}>批量重启</Button>
          </Space>
        )}

        {/* 表格 */}
        <Table
          columns={columns}
          dataSource={vms}
          rowKey="id"
          loading={loading}
          scroll={{ x: 1400 }}
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (page, size) => {
              setCurrent(page);
              setPageSize(size);
            },
          }}
        />
      </Space>

      {/* 创建虚拟机模态框 - 两列响应式布局 */}
      <Modal
        title="创建虚拟机"
        open={createModalVisible}
        onOk={handleCreate}
        onCancel={() => setCreateModalVisible(false)}
        confirmLoading={loading}
        width={800}
      >
        <Spin spinning={loading && !vtpPlatforms.length} tip="加载VDI配置中...">
          <Form form={form} layout="vertical">
            <Row gutter={16}>
              {/* 左列：基础配置 */}
              <Col xs={24} md={12}>
                <Form.Item label="VDI 服务器" name="vdi_server_id" rules={[{ required: true, message: '请选择VDI服务器' }]}>
                  <Select placeholder="请选择VDI服务器" loading={vdiServers.length === 0}>
                    {vdiServers.filter(s => s.status === 0).map(server => (
                      <Select.Option key={server.id} value={server.id}>
                        {server.name}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>

                <Form.Item label="资源组" name="resource_group_id" rules={[{ required: true, message: '请选择资源组' }]}>
                  <Select placeholder={selectedServerId ? "请选择资源组" : "请先选择VDI服务器"} disabled={!selectedServerId}>
                    {resourceGroups.map(group => (
                      <Select.Option key={group.resource_group_id} value={group.resource_group_id}>
                        {group.name} {group.type ? `(${group.type})` : ''}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>

                <Form.Item label="资源" name="resource_id" rules={[{ required: true, message: '请选择资源' }]}>
                  <Select placeholder={selectedResourceGroupId ? "请选择资源" : "请先选择资源组"} disabled={!selectedResourceGroupId}>
                    {resources.map(r => (
                      <Select.Option key={r.id} value={String(r.id)}>
                        {r.name} {r.note ? `(${r.note})` : ''}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>

                <Form.Item label="VTP 平台" name="vtp_id" rules={[{ required: true, message: '请选择VTP平台' }]}>
                  <Select placeholder={selectedServerId ? "请选择VTP平台" : "请先选择VDI服务器"} disabled={!selectedServerId}>
                    {vtpPlatforms.map(platform => (
                      <Select.Option key={platform.id} value={platform.id}>
                        {platform.name}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>

                <Form.Item label="运行位置" name="run_position_id" rules={[{ required: true, message: '请选择运行位置' }]}>
                  <Select placeholder={selectedVtpId ? "请选择运行位置" : "请先选择VTP平台"} disabled={!selectedVtpId}>
                    {runPositions.map(position => (
                      <Select.Option key={position.id} value={position.id}>
                        {position.name}
                      </Select.Option>
                    ))}
                  </Select>
                  <div style={{ fontSize: '12px', color: '#999', marginTop: '4px' }}>
                    服务器会自动分配到 10.62.0.73 或 10.62.0.74
                  </div>
                </Form.Item>

                <Form.Item label="个人盘" name="disk_id" rules={[{ required: true, message: '请选择个人盘' }]}>
                  <Select placeholder={selectedVtpId ? "请选择个人盘" : "请先选择VTP平台"} disabled={!selectedVtpId}>
                    {storages.map(storage => (
                      <Select.Option key={storage.id} value={storage.id}>
                        {storage.name} ({(parseInt(storage.avail) / 1024).toFixed(1)}GB可用)
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>

                <Form.Item label="CPU 颗数" name="cpu_number" initialValue={1}>
                  <Slider min={1} max={16} marks={{ 1: '1', 4: '4', 8: '8', 16: '16' }} />
                </Form.Item>
              </Col>

              {/* 右列：虚拟机配置 */}
              <Col xs={24} md={12}>
                <Form.Item label="存储位置" name="storage_id" rules={[{ required: true, message: '请选择存储位置' }]}>
                  <Select placeholder={selectedVtpId ? "请选择存储位置" : "请先选择VTP平台"} disabled={!selectedVtpId}>
                    {storages.map(storage => (
                      <Select.Option key={storage.id} value={storage.id}>
                        {storage.name} ({storage.avail}MB可用)
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>

                <Form.Item label="网络接口" name="network_id" rules={[{ required: true, message: '请选择网络接口' }]}>
                  <Select placeholder={selectedVtpId ? "请选择网络接口" : "请先选择VTP平台"} disabled={!selectedVtpId}>
                    {networks.map(network => (
                      <Select.Option key={network.id} value={network.id}>
                        {network.name}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>

                <Form.Item label="虚拟机名称" name="name" rules={[{ required: true, message: '请输入虚拟机名称' }]}>
                  <Input placeholder="选择资源组后自动生成：VDI-资源组-资源-后缀" />
                </Form.Item>

                <Form.Item label="名称后缀（可选）" name="name_suffix">
                  <Input
                    placeholder="自定义后缀，例如：zhangsan-001"
                    onChange={(e) => {
                      const suffix = e.target.value;
                      const selectedGroup = form.getFieldValue('resource_group_id');
                      const selectedResource = form.getFieldValue('resource_id');
                      if (selectedGroup && selectedResource) {
                        const group = resourceGroups.find(g => g.resource_group_id === selectedGroup);
                        const resource = resources.find(r => String(r.id) === selectedResource);
                        if (group && resource) {
                          const baseName = `VDI-${group.name}-${resource.name}`;
                          form.setFieldsValue({
                            name: suffix ? `${baseName}-${suffix}` : baseName
                          });
                        }
                      }
                    }}
                  />
                </Form.Item>

                <Form.Item label="创建数量" name="count" initialValue={1} rules={[{ required: true, message: '请输入创建数量' }]}>
                  <InputNumber min={1} max={10} style={{ width: '100%' }} />
                </Form.Item>

                <Form.Item label="主机位置" name="host_id">
                  <Input disabled placeholder="自动从运行位置获取 (father_id)" />
                </Form.Item>

                <Form.Item label="内存" name="memory" initialValue={4096}>
                  <Slider
                    min={512}
                    max={65536}
                    step={512}
                    marks={{ 4096: '4GB', 8192: '8GB', 16384: '16GB', 32768: '32GB' }}
                    tooltip={{ formatter: (v) => v ? `${(v / 1024).toFixed(1)} GB` : '' }}
                  />
                </Form.Item>
              </Col>
            </Row>

            {/* 全宽行：CPU核数和磁盘 */}
            <Row gutter={16}>
              <Col xs={24} md={12}>
                <Form.Item label="每颗 CPU 核数" name="cpu_core" initialValue={4}>
                  <Slider min={1} max={32} marks={{ 1: '1', 8: '8', 16: '16', 32: '32' }} />
                </Form.Item>
              </Col>
              <Col xs={24} md={12}>
                <Form.Item label="磁盘 (GB)" name="disk" initialValue={60}>
                  <Slider
                    min={20}
                    max={500}
                    step={10}
                    marks={{ 20: '20GB', 100: '100GB', 200: '200GB', 500: '500GB' }}
                    tooltip={{ formatter: (v) => v ? `${v} GB` : '' }}
                  />
                </Form.Item>
              </Col>
            </Row>
          </Form>
        </Spin>
      </Modal>

      {/* 配置 IP 模态框 */}
      <Modal
        title="配置 IP 地址"
        open={configIPModalVisible}
        onOk={handleConfigIP}
        onCancel={() => setConfigIPModalVisible(false)}
        confirmLoading={loading}
      >
        <Form form={form} layout="vertical">
          <Form.Item label="IP 地址" name="ip_address" rules={[{ required: true }]}>
            <Input placeholder="192.168.1.100" />
          </Form.Item>
          <Form.Item label="子网掩码" name="netmask" rules={[{ required: true }]}>
            <Input placeholder="255.255.255.0" />
          </Form.Item>
          <Form.Item label="网关" name="gateway" rules={[{ required: true }]}>
            <Input placeholder="192.168.1.1" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 重命名模态框 */}
      <Modal
        title="重命名虚拟机"
        open={renameModalVisible}
        onOk={handleRename}
        onCancel={() => setRenameModalVisible(false)}
        confirmLoading={loading}
      >
        <Form form={form} layout="vertical">
          <Form.Item label="新名称" name="new_name" rules={[{ required: true }]}>
            <Input placeholder="新虚拟机名称" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 绑定用户模态框 */}
      <Modal
        title="绑定用户"
        open={bindUserModalVisible}
        onOk={handleBindUser}
        onCancel={() => setBindUserModalVisible(false)}
        confirmLoading={loading}
      >
        <Form form={form} layout="vertical">
          <Form.Item label="用户 ID" name="user_id" rules={[{ required: true }]}>
            <Input placeholder="用户 ID" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
};

export default VirtualMachineList;