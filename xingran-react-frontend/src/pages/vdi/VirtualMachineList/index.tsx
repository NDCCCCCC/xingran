import React, { useState, useEffect, useCallback, useRef } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import { useAuthStore } from "@/store/authStore";
import { useMenuStore } from "@/store/menuStore";
import { vmOperationButtons } from "./vmOperationButtons";
import {
  Table,
  Button,
  Space,
  Tag,
  App,
  Modal,
  Select,
  Card,
  Form,
  Slider,
  InputNumber,
  Input,
  Row,
  Col,
  Spin,
  Alert,
} from "antd";
import {
  ReloadOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import { post } from "@/lib/api";
import { vmApi, vdiServerApi } from "@/lib/vdiApi";
import type { VirtualMachine, VMListParams, CreateVMRequest, VDIServer, VDIResourceGroup, VDIResource, VDIPlatform, RunPosition, VDIStorage, VDINetwork } from "@/types/vdi";
import type { ColumnsType } from "antd/es/table";
import { VDIRow } from "@/components/table/VDIRow";

// Permission helper
const hasPermission = (permissions: string[], perm: string) => permissions.includes(perm);

const VirtualMachineList: React.FC = () => {
  const navigate = useNavigate();
  const { user: _user } = useAuthStore();
  // Use permissions from menuStore (loaded via /system/my-menus/permissions)
  // authStore user.permissions is NOT populated from the login API
  const menuPermissions = useMenuStore(state => state.permissions);
  const permissions = menuPermissions;
  const [loading, setLoading] = useState(false);
  const [vms, setVMs] = useState<VirtualMachine[]>([]);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const location = useLocation();
  const [filters, setFilters] = usePersistedStateController<Partial<VMListParams>>({
    keyPrefix: location.pathname,
    keySuffix: "filters",
    defaultValue: {},
  });
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [quickCreateModalVisible, setQuickCreateModalVisible] = useState(false);
  const [bindUserModalVisible, setBindUserModalVisible] = useState(false);
  const [selectedVM, setSelectedVM] = useState<VirtualMachine | null>(null);
  const [form] = Form.useForm();
  const [quickCreateForm] = Form.useForm();
  const [bindUserForm] = Form.useForm();
  const { message } = App.useApp();

  // Bind user modal: system user search state
  const [systemUsers, setSystemUsers] = useState<Array<{id: string; username: string; nickname?: string}>>([]);
  const [userSearchLoading, setUserSearchLoading] = useState(false);
  const userSearchTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  // 快速创建默认配置状态
  const [quickConfig, setQuickConfig] = useState<{
    vdiServerId: string;
    resourceId: string;
    runPositionId: string;
    diskId: string;
    storageId: string;
    networkId: string;
    hostId: string;
    vtpId: number;
    resourceGroupId: string;
    resourceGroupName: string;
  } | null>(null);
  const [vdiServers, setVdiServers] = useState<VDIServer[]>([]);
  const [resourceGroups, setResourceGroups] = useState<VDIResourceGroup[]>([]);
  const [resources, setResources] = useState<VDIResource[]>([]);
  const [vtpPlatforms, setVtpPlatforms] = useState<VDIPlatform[]>([]);
  const [runPositions, setRunPositions] = useState<RunPosition[]>([]);
  const [storages, setStorages] = useState<VDIStorage[]>([]);
  const [networks, setNetworks] = useState<VDINetwork[]>([]);

  // VDI数据预加载状态
  const [vdiDataPreloaded, setVdiDataPreloaded] = useState(false);
  const [_vdiDataLoading, setVdiDataLoading] = useState(false);
  const [preloadError, setPreloadError] = useState<string | null>(null);

  // Whether user has VM create permission (controls preload and toolbar buttons)
  const canCreateVM = hasPermission(permissions, "vdi:vm:add");

  // API缓存：5分钟过期
  const vdiDataCache = useRef<{
    vtpPlatforms: VDIPlatform[];
    runPositions: RunPosition[];
    storages: VDIStorage[];
    networks: VDINetwork[];
    timestamp: number;
  } | null>(null);
  const CACHE_DURATION = 5 * 60 * 1000; // 5分钟

  // 预加载VDI配置数据（仅在用户有 vdi:vm:add 权限时）
  const preloadVDIData = useCallback(async () => {
    // Skip preload if user lacks create permission
    if (!canCreateVM) {
      return;
    }

    // 检查缓存
    if (vdiDataCache.current && (Date.now() - vdiDataCache.current.timestamp < CACHE_DURATION)) {
      setVdiDataPreloaded(true);
      return;
    }

    setVdiDataLoading(true);

    try {
      // 获取第一个可用的VDI服务器
      const serverResult = await vdiServerApi.list({ current: 1, pageSize: 100 });
      const servers = serverResult.data?.list || [];
      const availableServer = servers.find(s => s.status === 0);

      if (!availableServer) {
        setVdiDataLoading(false);
        return;
      }

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
    } catch (_error) {
      setPreloadError("VDI配置加载失败，部分功能可能不可用");
      message.warning("VDI配置加载失败，将在需要时重试");
    } finally {
      setVdiDataLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [canCreateVM, CACHE_DURATION]);

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
    } catch (_error) {
      message.error("加载虚拟机列表失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadVMs();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [current, pageSize, filters]);

  // 页面加载时预加载VDI配置数据（仅在用户有创建权限时）
  useEffect(() => {
    const timer = setTimeout(() => {
      preloadVDIData();
    }, 100);
    return () => clearTimeout(timer);
  }, [preloadVDIData]);

  // Watch form field changes for cascading dropdowns and auto-name
  const selectedServerId = Form.useWatch("vdi_server_id", form);
  const selectedResourceGroupId = Form.useWatch("resource_group_id", form);
  const _selectedResourceFieldId = Form.useWatch("resource_id", form);
  const selectedVtpId = Form.useWatch("vtp_id", form);

  // Track previous createModalVisible value to prevent infinite loop
  const prevCreateModalVisible = useRef(createModalVisible);

  // Load resource groups when VDI server is selected
  useEffect(() => {
    // Only trigger on transition from false to true or when server changes
    if (selectedServerId && createModalVisible && !prevCreateModalVisible.current) {
      vmApi.listResourceGroups(selectedServerId).then(result => {
        setResourceGroups(result.data || []);
      }).catch(() => {
        setResourceGroups([]);
      });
      // Clear resources when VDI server changes
      setResources([]);
      form.setFieldsValue({ resource_group_id: undefined, resource_id: undefined });
    } else if (!createModalVisible) {
      setResourceGroups([]);
    }

    prevCreateModalVisible.current = createModalVisible;
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
  }, [selectedServerId, createModalVisible, form, CACHE_DURATION]);

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
      const _loadPositions = () => {
        if (useCache && vdiDataCache.current!.runPositions.length > 0) {
          setRunPositions(vdiDataCache.current!.runPositions);
          positionsLoaded = true;
        } else {
          return vmApi.listRunPositions(selectedServerId, selectedVtpId);
        }
      };

      // Load storages
      const _loadStorages = () => {
        if (useCache && vdiDataCache.current!.storages.length > 0) {
          setStorages(vdiDataCache.current!.storages);
          storagesLoaded = true;
        } else {
          return vmApi.listStorages(selectedServerId, selectedVtpId);
        }
      };

      // Load networks
      const _loadNetworks = () => {
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
        const clusteredHostId = newPositions.find(p =>
          p.father_id && p.father_id.toLowerCase().includes("clustered")
        )?.father_id;

        if (clusteredHostId) {
          newPositions = newPositions.filter(p => p.father_id === clusteredHostId);
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
  }, [selectedVtpId, selectedServerId, createModalVisible, form, vtpPlatforms, CACHE_DURATION]);

  // Open create modal and load dropdown data (使用预加载数据)
  const openCreateModal = async () => {
    setCreateModalVisible(true);
    form.resetFields();

    // 如果已有预加载数据，直接使用
    if (vdiDataPreloaded && vdiDataCache.current) {
      setVtpPlatforms(vdiDataCache.current.vtpPlatforms);
      setRunPositions(vdiDataCache.current.runPositions);
      setStorages(vdiDataCache.current.storages);
      setNetworks(vdiDataCache.current.networks);

      // 加载VDI服务器列表（这个还是需要调用API）
      try {
        const serverResult = await vdiServerApi.list({ current: 1, pageSize: 100 });
        setVdiServers(serverResult.data?.list || []);
      } catch (_e) {
        // ignore
      }
      return;
    }

    // 如果没有预加载数据，显示加载状态并开始加载
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
        form.setFieldsValue({ vdi_server_id: availableServer.id });
      }
    } catch (_e) {
      // ignore
    }
  };

  // 创建虚拟机（调用 VDI API）
  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      let hostId = values.host_id;
      let runPositionId = values.run_position_id;

      if (values.run_position_id) {
        const selectedPosition = runPositions.find(p => p.id === values.run_position_id);
        if (selectedPosition) {
          hostId = selectedPosition.father_id;
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
      message.success("虚拟机创建成功，VDI API 调用完成");
      setCreateModalVisible(false);
      form.resetFields();
      loadVMs();
    } catch (_error) {
      message.error("创建虚拟机失败");
    } finally {
      setLoading(false);
    }
  };

  // 操作虚拟机（调用 VDI API）
  const handleOperate = useCallback(async (action: string, vmIds?: string[]) => {
    const ids = vmIds || selectedRowKeys as string[];
    if (ids.length === 0) {
      message.warning("请选择要操作的虚拟机");
      return;
    }

    setLoading(true);
    try {
      await vmApi.batchOperate(ids, action);
      message.success(`${action === "start" ? "开机" : action === "stop" ? "关机" : "重启"}操作已提交，VDI API 调用成功`);
      setSelectedRowKeys([]);
      loadVMs();
    } catch (_error) {
      message.error("操作失败，VDI API 调用失败");
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedRowKeys, loadVMs]);

  // 删除虚拟机（调用 VDI API）
  const handleDelete = useCallback(async (id: string) => {
    setLoading(true);
    try {
      await vmApi.delete(id);
      message.success("删除成功，VDI API 调用完成");
      loadVMs();
    } catch (_error) {
      message.error("删除失败，VDI API 调用失败");
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loadVMs]);

  // 同步虚拟机（调用 VDI API）
  const handleSync = useCallback(async (id: string) => {
    setLoading(true);
    try {
      await vmApi.sync(id);
      message.success("同步成功，VDI 状态已更新");
      loadVMs();
    } catch (_error) {
      message.error("同步失败");
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loadVMs]);

  // 绑定用户操作
  const handleBind = useCallback((vm: VirtualMachine) => {
    setSelectedVM(vm);
    setBindUserModalVisible(true);
  }, []);

  // Load system users for bind user modal
  const loadSystemUsers = async (search?: string) => {
    setUserSearchLoading(true);
    try {
      const result = await post<{ list: Array<{id: string; username: string; nickname?: string; status: number}>; total: number }>("/system/users/list", {
        current: 1,
        pageSize: 50,
        username: search || undefined,
        nickname: search || undefined,
      });
      const users = (result.data?.list || []).filter(u => u.status === 0);
      setSystemUsers(users);
    } catch {
      setSystemUsers([]);
    } finally {
      setUserSearchLoading(false);
    }
  };

  // 绑定用户（调用 VDI API）
  const handleBindUser = async () => {
    try {
      const values = await bindUserForm.validateFields();
      setLoading(true);
      await vmApi.bindUser(selectedVM!.id, { username: values.username });
      message.success("用户绑定成功，VDI API 调用完成");
      setBindUserModalVisible(false);
      bindUserForm.resetFields();
      loadVMs();
    } catch (_error) {
      message.error("绑定用户失败，VDI API 调用失败");
    } finally {
      setLoading(false);
    }
  };

  // ============================================================================
  // 快速创建功能 - 自动加载默认配置
  // ============================================================================
  const DEFAULT_VTP_ID = 1;
  const DEFAULT_RESOURCE_GROUP_ID = "0";
  const DEFAULT_RESOURCE_NAME = "VDI模板";
  const DEFAULT_POSITION_NAME = "研发";

  // 加载快速创建默认配置
  const loadQuickCreateDefaults = async () => {
    try {
      const serverResult = await vdiServerApi.list({ current: 1, pageSize: 100 });
      const servers = serverResult.data?.list || [];
      const availableServer = servers.find(s => s.status === 0);

      if (!availableServer) {
        message.error("没有可用的VDI服务器");
        return;
      }

      const resourceGroups = await vmApi.listResourceGroups(availableServer.id);
      const defaultGroup = resourceGroups.data?.find(g => g.resource_group_id === DEFAULT_RESOURCE_GROUP_ID);

      if (!defaultGroup) {
        message.error("未找到默认资源组");
        return;
      }

      const resources = await vmApi.listResources(availableServer.id, DEFAULT_RESOURCE_GROUP_ID);
      const resourceList = resources.data || [];
      const dataResource = resourceList.find(r => r.name === DEFAULT_RESOURCE_NAME);

      if (!dataResource) {
        message.error(`未找到资源"${DEFAULT_RESOURCE_NAME}"，可用资源: ${resourceList.map(r => r.name).join(", ")}`);
        return;
      }

      const platforms = await vmApi.listVTPPlatforms(availableServer.id);
      const vmpPlatform = platforms.data?.find(p => p.name === "VMP" || p.id === DEFAULT_VTP_ID);

      if (!vmpPlatform) {
        message.error("未找到VMP平台");
        return;
      }

      const positions = await vmApi.listRunPositions(availableServer.id, vmpPlatform.id);
      const positionList = positions.data || [];
      const devPosition = positionList.find(p => p.name === DEFAULT_POSITION_NAME);

      if (!devPosition) {
        message.error(`未找到运行位置"${DEFAULT_POSITION_NAME}"，可用位置: ${positionList.map(p => p.name).join(", ")}`);
        return;
      }

      const storages = await vmApi.listStorages(availableServer.id, vmpPlatform.id);
      const storageList = storages.data || [];
      const firstStorage = storageList[0];

      if (!firstStorage) {
        message.error("未找到存储位置");
        return;
      }

      const networks = await vmApi.listNetworks(availableServer.id, vmpPlatform.id);
      const networkList = networks.data || [];
      const firstNetwork = networkList[0];

      if (!firstNetwork) {
        message.error("未找到网络接口");
        return;
      }

      setQuickConfig({
        vdiServerId: availableServer.id,
        resourceId: String(dataResource.id),
        runPositionId: devPosition.id,
        diskId: firstStorage.id,
        storageId: firstStorage.id,
        networkId: firstNetwork.id,
        hostId: devPosition.father_id || devPosition.id,
        vtpId: vmpPlatform.id,
        resourceGroupId: DEFAULT_RESOURCE_GROUP_ID,
        resourceGroupName: defaultGroup.name,
      });

      quickCreateForm.setFieldsValue({
        count: 1,
      });

      message.success("快速创建配置加载成功！");
    } catch (_error) {
      message.error("加载VDI配置失败，请检查网络连接和VDI服务器状态");
    }
  };

  // 打开快速创建模态框
  const openQuickCreateModal = async () => {
    setQuickCreateModalVisible(true);
    quickCreateForm.resetFields();
    await loadQuickCreateDefaults();
  };

  // 快速创建虚拟机
  const handleQuickCreate = async () => {
    if (!quickConfig) {
      message.error("配置未加载，请重试");
      return;
    }

    try {
      const values = await quickCreateForm.validateFields();
      setLoading(true);

      let finalRunPositionId = quickConfig.runPositionId;
      if (quickConfig.runPositionId === quickConfig.hostId) {
        finalRunPositionId = "";
      }

      const data: CreateVMRequest = {
        name: values.name,
        vdi_server_id: quickConfig.vdiServerId,
        resource_group_id: quickConfig.resourceGroupId,
        resource_id: quickConfig.resourceId,
        vtp_id: quickConfig.vtpId,
        host_id: quickConfig.hostId,
        run_position_id: finalRunPositionId,
        disk_id: quickConfig.diskId,
        storage_id: quickConfig.storageId,
        network_id: quickConfig.networkId,
        count: values.count,
      };

      await vmApi.create(data);
      message.success("虚拟机创建成功！");
      setQuickCreateModalVisible(false);
      quickCreateForm.resetFields();
      loadVMs();
    } catch (_error) {
      message.error("创建虚拟机失败");
    } finally {
      setLoading(false);
    }
  };

  // 表格列定义（操作按钮使用 VDIRow 组件进行权限过滤）
  const columns: ColumnsType<VirtualMachine> = [
    {
      title: "虚拟机 ID",
      dataIndex: "vm_id",
      key: "vm_id",
      width: 120,
      ellipsis: true,
    },
    {
      title: "名称",
      dataIndex: "name",
      key: "name",
      width: 150,
      render: (text: string, record) => (
        <a
          onClick={() => navigate(`/vdi/vm/${record.id}`)}
          style={{ color: "var(--theme-info, #1890ff)", textDecoration: "underline" }}
        >
          {text}
        </a>
      ),
    },
    {
      title: "虚拟机状态",
      dataIndex: "power_state",
      key: "power_state",
      width: 100,
      render: (state: string) => {
        const config = {
          pending: { color: "blue", text: "等待Agent" },
          stopped: { color: "red", text: "关机" },
          suspended: { color: "orange", text: "挂起" },
          in_use: { color: "lime", text: "正常使用" },
        };
        const { color, text } = config[state as keyof typeof config] || { color: "default", text: state };
        return <Tag color={color}>{text}</Tag>;
      },
    },
    {
      title: "IP 地址",
      dataIndex: "ip_address",
      key: "ip_address",
      width: 120,
      render: (ip: string) => ip || "-",
    },
    {
      title: "绑定用户",
      dataIndex: "bound_user_name",
      key: "bound_user_name",
      width: 120,
      render: (name: string) => name || "-",
    },
    {
      title: "最后同步",
      dataIndex: "last_sync_at",
      key: "last_sync_at",
      width: 160,
      render: (time: string) => time ? new Date(time).toLocaleString("zh-CN") : "-",
    },
    {
      title: "操作",
      key: "action",
      width: 300,
      fixed: "right",
      render: (_, record) => (
        <VDIRow
          vm={record}
          permissions={permissions}
          buttons={vmOperationButtons}
          onOperate={handleOperate}
          onDelete={handleDelete}
          onSync={handleSync}
          onBind={handleBind}
        />
      ),
    },
  ];

  return (
    <Card>
      <Space orientation="vertical" size="large" style={{ width: "100%" }}>
        {/* Error banner for VDI data preload failures (only shown when user has create permission) */}
        {preloadError && canCreateVM && (
          <Alert
            type="warning"
            title={preloadError}
            closable
            onClose={() => setPreloadError(null)}
            style={{ marginBottom: 16 }}
          />
        )}

        {/* 工具栏 */}
        <Space>
          {canCreateVM && (
            <>
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreateModal}>
                创建虚拟机
              </Button>
              <Button onClick={openQuickCreateModal}>
                快速创建
              </Button>
            </>
          )}
          <Input.Search
            placeholder="搜索虚拟机名称"
            style={{ width: 200 }}
            onSearch={(value) => setFilters({ ...filters, name: value })}
          />
          <Select
            placeholder="虚拟机状态"
            style={{ width: 120 }}
            allowClear
            onChange={(value) =>    setFilters({ ...filters, powerState: value })}
           onSearch={() => {}}>
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
            {hasPermission(permissions, "vdi:vm:start") && <Button onClick={() => handleOperate("start")}>批量开机</Button>}
            {hasPermission(permissions, "vdi:vm:stop") && <Button onClick={() => handleOperate("stop")}>批量关机</Button>}
            {hasPermission(permissions, "vdi:vm:restart") && <Button onClick={() => handleOperate("restart")}>批量重启</Button>}
          </Space>
        )}

        {/* 表格 */}
        <Table
          columns={columns}
          dataSource={vms}
          rowKey="id"
          loading={loading}
          virtual
          scroll={{ x: 1600, y: 600 }}
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
            columnWidth: 48,
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
        <Spin spinning={loading && !vtpPlatforms.length}>
          {loading && !vtpPlatforms.length && (
            <div style={{ padding: 12, color: "rgba(0, 0, 0, 0.45)" }}>加载VDI配置中...</div>
          )}
          <Form form={form} layout="vertical">
            <Row gutter={16}>
              {/* 左列：基础配置 */}
              <Col xs={24} md={12}>
                <Form.Item label="VDI 服务器" name="vdi_server_id" rules={[{ required: true, message: "请选择VDI服务器" }]}>
                  <Select placeholder="请选择VDI服务器" loading={vdiServers.length === 0} onSearch={() => {}}>
                    {vdiServers.filter(s => s.status === 0).map(server => (
                      <Select.Option key={server.id} value={server.id}>
                        {server.name}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>

                <Form.Item label="资源组" name="resource_group_id" rules={[{ required: true, message: "请选择资源组" }]}>
                  <Select placeholder={selectedServerId ? "请选择资源组" : "请先选择VDI服务器"} disabled={!selectedServerId} onSearch={() => {}}>
                    {resourceGroups.map(group => (
                      <Select.Option key={group.resource_group_id} value={group.resource_group_id}>
                        {group.name} {group.type ? `(${group.type})` : ""}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>

                <Form.Item label="资源" name="resource_id" rules={[{ required: true, message: "请选择资源" }]}>
                  <Select placeholder={selectedResourceGroupId ? "请选择资源" : "请先选择资源组"} disabled={!selectedResourceGroupId} onSearch={() => {}}>
                    {resources.map(r => (
                      <Select.Option key={r.id} value={String(r.id)}>
                        {r.name} {r.note ? `(${r.note})` : ""}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>

                <Form.Item label="VTP 平台" name="vtp_id" rules={[{ required: true, message: "请选择VTP平台" }]}>
                  <Select placeholder={selectedServerId ? "请选择VTP平台" : "请先选择VDI服务器"} disabled={!selectedServerId} onSearch={() => {}}>
                    {vtpPlatforms.map(platform => (
                      <Select.Option key={platform.id} value={platform.id}>
                        {platform.name}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>

                <Form.Item label="运行位置" name="run_position_id" rules={[{ required: true, message: "请选择运行位置" }]}>
                  <Select placeholder={selectedVtpId ? "请选择运行位置" : "请先选择VTP平台"} disabled={!selectedVtpId} onSearch={() => {}}>
                    {runPositions.map(position => (
                      <Select.Option key={position.id} value={position.id}>
                        {position.name}
                      </Select.Option>
                    ))}
                  </Select>
                  <div style={{ fontSize: "12px", color: "var(--theme-text-tertiary, #999)", marginTop: "4px" }}>
                    服务器会自动分配到 10.62.0.73 或 10.62.0.74
                  </div>
                </Form.Item>

                <Form.Item label="个人盘" name="disk_id" rules={[{ required: true, message: "请选择个人盘" }]}>
                  <Select placeholder={selectedVtpId ? "请选择个人盘" : "请先选择VTP平台"} disabled={!selectedVtpId} onSearch={() => {}}>
                    {storages.map(storage => (
                      <Select.Option key={storage.id} value={storage.id}>
                        {storage.name} ({(parseInt(storage.avail) / 1024).toFixed(1)}GB可用)
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>

                <Form.Item label="CPU 颗数" name="cpu_number" initialValue={1}>
                  <Slider min={1} max={16} marks={{ 1: "1", 4: "4", 8: "8", 16: "16" }} />
                </Form.Item>
              </Col>

              {/* 右列：虚拟机配置 */}
              <Col xs={24} md={12}>
                <Form.Item label="存储位置" name="storage_id" rules={[{ required: true, message: "请选择存储位置" }]}>
                  <Select placeholder={selectedVtpId ? "请选择存储位置" : "请先选择VTP平台"} disabled={!selectedVtpId} onSearch={() => {}}>
                    {storages.map(storage => (
                      <Select.Option key={storage.id} value={storage.id}>
                        {storage.name} ({storage.avail}MB可用)
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>

                <Form.Item label="网络接口" name="network_id" rules={[{ required: true, message: "请选择网络接口" }]}>
                  <Select placeholder={selectedVtpId ? "请选择网络接口" : "请先选择VTP平台"} disabled={!selectedVtpId} onSearch={() => {}}>
                    {networks.map(network => (
                      <Select.Option key={network.id} value={network.id}>
                        {network.name}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>

                <Form.Item label="创建数量" name="count" initialValue={1} rules={[{ required: true, message: "请输入创建数量" }]}>
                  <InputNumber min={1} max={10} style={{ width: "100%" }} />
                </Form.Item>

                <Form.Item label="主机位置" name="host_id">
                  <Input disabled placeholder="自动从运行位置获取 (father_id)" />
                </Form.Item>

                <Form.Item label="内存" name="memory" initialValue={4096}>
                  <Slider
                    min={512}
                    max={65536}
                    step={512}
                    marks={{ 4096: "4GB", 8192: "8GB", 16384: "16GB", 32768: "32GB" }}
                    tooltip={{ formatter: (v) => v ? `${(v / 1024).toFixed(1)} GB` : "" }}
                  />
                </Form.Item>
              </Col>
            </Row>

            {/* 全宽行：CPU核数和磁盘 */}
            <Row gutter={16}>
              <Col xs={24} md={12}>
                <Form.Item label="每颗 CPU 核数" name="cpu_core" initialValue={4}>
                  <Slider min={1} max={32} marks={{ 1: "1", 8: "8", 16: "16", 32: "32" }} />
                </Form.Item>
              </Col>
              <Col xs={24} md={12}>
                <Form.Item label="磁盘 (GB)" name="disk" initialValue={60}>
                  <Slider
                    min={20}
                    max={500}
                    step={10}
                    marks={{ 20: "20GB", 100: "100GB", 200: "200GB", 500: "500GB" }}
                    tooltip={{ formatter: (v) => v ? `${v} GB` : "" }}
                  />
                </Form.Item>
              </Col>
            </Row>
          </Form>
        </Spin>
      </Modal>

      {/* 快速创建虚拟机模态框 */}
      <Modal
        title="快速创建虚拟机"
        open={quickCreateModalVisible}
        onOk={handleQuickCreate}
        onCancel={() => setQuickCreateModalVisible(false)}
        confirmLoading={loading}
        width={600}
      >
        <Spin spinning={loading && !quickConfig}>
          {loading && !quickConfig && (
            <div style={{ padding: 12, color: "rgba(0, 0, 0, 0.45)" }}>加载配置中...</div>
          )}
          <Form form={quickCreateForm} layout="vertical">
            {quickConfig && (
              <div style={{ marginBottom: 16, padding: 12, background: "#f5f5f5", borderRadius: 4 }}>
                <div><strong>已加载的默认配置：</strong></div>
                <div style={{ marginTop: 8, fontSize: "12px", color: "var(--theme-text-tertiary, #666)" }}>
                  • VDI 服务器: 自动选择可用服务器<br />
                  • 资源组: {quickConfig.resourceGroupName}<br />
                  • 虚拟机: {DEFAULT_RESOURCE_NAME}<br />
                  • VTP 平台: VMP<br />
                  • 运行位置: {DEFAULT_POSITION_NAME}<br />
                  • 存储/网络: 自动选择第一个可用
                </div>
              </div>
            )}

            <Form.Item label="创建数量" name="count" initialValue={1} rules={[{ required: true }]}>
              <InputNumber min={1} max={10} style={{ width: "100%" }} />
            </Form.Item>
          </Form>
        </Spin>
      </Modal>

      {/* 绑定用户模态框 */}
      <Modal
        title="绑定用户"
        open={bindUserModalVisible}
        onOk={handleBindUser}
        onCancel={() => { setBindUserModalVisible(false); bindUserForm.resetFields(); }}
        confirmLoading={loading}
      >
        <Form form={bindUserForm} layout="vertical">
          <Form.Item label="用户" name="username" rules={[{ required: true, message: "请选择用户" }]}>
            <Select
              showSearch
              placeholder="请输入用户名或昵称搜索"
              filterOption={false}
              loading={userSearchLoading}
              onSearch={(value) => {
                if (userSearchTimer.current) clearTimeout(userSearchTimer.current);
                userSearchTimer.current = setTimeout(() => loadSystemUsers(value), 300);
              }}
              onFocus={() => {
                if (systemUsers.length === 0) {
                  loadSystemUsers("");
                }
              }}
              notFoundContent={userSearchLoading ? "搜索中..." : "未找到用户"}
            >
              {systemUsers.map(user => (
                <Select.Option key={user.id} value={user.username}>
                  {user.nickname || user.username}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
};

export default VirtualMachineList;
