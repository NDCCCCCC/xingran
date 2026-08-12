---
phase: quick
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/api/v1/vdi/vm_handler.go
  - internal/api/v1/vdi/vm_router.go
  - internal/services/vdi/vm_service.go
  - internal/services/vdi/vm_service_impl.go
  - xingran-react-frontend/src/lib/vdiApi.ts
  - xingran-react-frontend/src/types/vdi.ts
  - xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx
autonomous: true
requirements: []
must_haves:
  truths:
    - "用户打开创建虚拟机表单时，资源组和VDI服务器显示为下拉框，展示名称而非ID"
    - "名称字段根据选中的资源组名称自动生成前缀，用户可添加自定义后缀"
    - "CPU颗数、内存、磁盘使用滑动条控件，带有数值标记和单位显示"
    - "选择VDI服务器后，资源组下拉框过滤为该服务器下的资源组"
  artifacts:
    - path: "internal/api/v1/vdi/vm_handler.go"
      provides: "ListResourceGroups handler endpoint"
    - path: "xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx"
      provides: "Redesigned create VM modal with dropdowns and sliders"
    - path: "xingran-react-frontend/src/lib/vdiApi.ts"
      provides: "Frontend API for fetching resource groups"
  key_links:
    - from: "VirtualMachineList/index.tsx"
      to: "/vdi/vm/resource-groups"
      via: "vdiApi.listResourceGroups fetch on VDI server change"
    - from: "VirtualMachineList/index.tsx"
      to: "/vdi/servers/list"
      via: "vdiServerApi.list fetch on modal open"
    - from: "vm_handler.go ListResourceGroups"
      to: "vm_service.go ListResourceGroupsByServer"
      via: "service call with vdi_server_id param"
---

<objective>
优化VDI虚拟机创建表单用户体验：名称根据模板（资源组）自动生成，资源组ID和VDI服务器ID改为下拉框选择（显示名称），CPU/内存/硬盘使用滑动条控件。

Purpose: 当前创建表单要求用户手动输入UUID和数值，体验差且容易出错。改为下拉选择和滑动条后，减少输入错误，提升操作效率。
Output: 改造后的创建虚拟机模态框 + 资源组列表API
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md
</context>

<interfaces>
<!-- Key types executor needs -->

From xingran-react-frontend/src/types/vdi.ts:
```typescript
export interface VDIServer {
  id: string;
  name: string;
  endpoint: string;
  username: string;
  tenant_id: number;
  status: number;
  token_expiry?: string;
  lastSyncTime?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateVMRequest {
  name: string;
  resource_id: string;
  vdi_server_id: string;
  cpu_number?: number;
  cpu_core?: number;
  memory?: number;
  disk?: number;
}
```

From xingran-react-frontend/src/lib/vdiApi.ts:
```typescript
export const vdiServerApi = {
  list: async (params: { current: number; pageSize: number }) => {
    return await post<PageResponse<VDIServer>>('/vdi/servers/list', params);
  },
};
```

From internal/models/vdi.go - VDIResourceGroup:
```go
type VDIResourceGroup struct {
    BaseModel
    ResourceGroupID string `json:"resource_group_id"`
    Name            string `json:"name"`
    VdiServerID     string `json:"vdi_server_id"`
    Type            string `json:"type"`
    Status          int    `json:"status"`
}
```

From internal/services/vdi/vm_service.go - CreateVMServiceRequest:
```go
type CreateVMServiceRequest struct {
    Name        string `json:"name" validate:"required"`
    ResourceID  string `json:"resource_id" validate:"required"`
    VdiServerID string `json:"vdi_server_id" validate:"required"`
    CPUNumber   int    `json:"cpu_number" validate:"min=1,max=64"`
    CPUCore     int    `json:"cpu_core" validate:"min=1,max=128"`
    Memory      int    `json:"memory" validate:"min=512,max=131072"`
    Disk        int    `json:"disk" validate:"min=20,max=2000"`
}
```
</interfaces>

<tasks>

<task type="auto">
  <name>Task 1: Add backend resource group list API endpoint</name>
  <files>internal/services/vdi/vm_service.go, internal/services/vdi/vm_service_impl.go, internal/api/v1/vdi/vm_handler.go, internal/api/v1/vdi/vm_router.go</files>
  <action>
1. In `internal/services/vdi/vm_service.go`, add to the VMService interface:
   ```go
   ListResourceGroups(ctx context.Context, vdiServerID string) ([]VDIResourceGroupDTO, error)
   ```

2. Add the DTO type in `vm_service.go`:
   ```go
   type VDIResourceGroupDTO struct {
       ResourceGroupID string `json:"resource_group_id"`
       Name            string `json:"name"`
       VdiServerID     string `json:"vdi_server_id"`
       Type            string `json:"type"`
   }
   ```

3. In `internal/services/vdi/vm_service_impl.go`, implement `ListResourceGroups`:
   - Query `sys_vdi_resource_group` from the local DB filtered by `vdi_server_id` and `status = 0` (enabled only)
   - If `vdiServerID` is empty, return all enabled resource groups
   - Map results to `VDIResourceGroupDTO` and return
   - Do NOT call the VDI external API for this -- the resource groups are already synced to local DB

4. In `internal/api/v1/vdi/vm_handler.go`, add a `ListResourceGroups` handler method:
   ```go
   func (h *VMHandler) ListResourceGroups(c *gin.Context) {
       var req struct {
           VdiServerID string `json:"vdi_server_id"`
       }
       if !handleJSONBinding(c, &req) {
           return
       }
       groups, err := h.vmService.ListResourceGroups(c.Request.Context(), req.VdiServerID)
       if !handleServiceError(c, err, "查询资源组") {
           return
       }
       response.Success(c, groups)
   }
   ```

5. In `internal/api/v1/vdi/vm_router.go`, register the new route:
   ```go
   r.POST("/resource-groups", vmHandler.ListResourceGroups)
   ```
   Add this BEFORE the `/:id` routes to avoid route conflicts.
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/xingran-go-backend && go build ./internal/api/v1/vdi/... ./internal/services/vdi/...</automated>
  </verify>
  <done>Backend build succeeds. New endpoint POST /vdi/vm/resource-groups returns resource groups filtered by vdi_server_id from local DB.</done>
</task>

<task type="auto">
  <name>Task 2: Redesign frontend create VM modal with dropdowns, sliders, and auto-name generation</name>
  <files>xingran-react-frontend/src/types/vdi.ts, xingran-react-frontend/src/lib/vdiApi.ts, xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx</files>
  <action>
1. In `xingran-react-frontend/src/types/vdi.ts`, add the resource group type:
   ```typescript
   export interface VDIResourceGroup {
     resource_group_id: string;
     name: string;
     vdi_server_id: string;
     type: string;
   }
   ```

2. In `xingran-react-frontend/src/lib/vdiApi.ts`:
   - Import the new `VDIResourceGroup` type
   - Add to `vmApi`:
     ```typescript
     listResourceGroups: async (vdiServerId?: string) => {
       return await post<VDIResourceGroup[]>('/vdi/vm/resource-groups', {
         vdi_server_id: vdiServerId || '',
       });
     },
     ```

3. In `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx`, redesign the create VM modal:

   a. Add imports: `Slider` from antd, remove `InputNumber` (or keep for fallback)

   b. Add state variables at the component level:
   ```typescript
   const [vdiServers, setVdiServers] = useState<VDIServer[]>([]);
   const [resourceGroups, setResourceGroups] = useState<VDIResourceGroup[]>([]);
   ```

   c. Add data loading when create modal opens. Modify the "创建虚拟机" button's onClick to also load dropdown data:
   ```typescript
   const openCreateModal = async () => {
     setCreateModalVisible(true);
     form.resetFields();
     // Load VDI servers
     try {
       const serverResult = await vdiServerApi.list({ current: 1, pageSize: 100 });
       setVdiServers(serverResult.data?.list || []);
     } catch (e) { /* ignore */ }
   };
   ```

   d. Add effect to load resource groups when VDI server is selected:
   ```typescript
   // Watch vdi_server_id changes in the form
   const selectedServerId = Form.useWatch('vdi_server_id', form);
   useEffect(() => {
     if (selectedServerId && createModalVisible) {
       vmApi.listResourceGroups(selectedServerId).then(result => {
         setResourceGroups(result.data || []);
       }).catch(() => {
         setResourceGroups([]);
       });
     }
   }, [selectedServerId, createModalVisible]);
   ```

   e. Add auto-name generation when resource group is selected:
   ```typescript
   const selectedResourceId = Form.useWatch('resource_id', form);
   useEffect(() => {
     if (selectedResourceId) {
       const group = resourceGroups.find(g => g.resource_group_id === selectedResourceId);
       if (group) {
         const suffix = form.getFieldValue('name_suffix') || '';
         form.setFieldsValue({ name: group.name + (suffix ? '-' + suffix : '') });
       }
     }
   }, [selectedResourceId, resourceGroups]);

   // Also update name when suffix changes
   const handleSuffixChange = (suffix: string) => {
     const selectedResource = form.getFieldValue('resource_id');
     const group = resourceGroups.find(g => g.resource_group_id === selectedResource);
     if (group) {
       form.setFieldsValue({ name: group.name + (suffix ? '-' + suffix : '') });
     }
   };
   ```

   f. Replace the create VM modal Form (lines 483-505) with the new design:
   ```jsx
   <Form form={form} layout="vertical">
     {/* VDI Server dropdown */}
     <Form.Item label="VDI 服务器" name="vdi_server_id" rules={[{ required: true, message: '请选择VDI服务器' }]}>
       <Select placeholder="请选择VDI服务器" loading={vdiServers.length === 0}>
         {vdiServers.filter(s => s.status === 0).map(server => (
           <Select.Option key={server.id} value={server.id}>
             {server.name}
           </Select.Option>
         ))}
       </Select>
     </Form.Item>

     {/* Resource Group dropdown (filtered by selected VDI server) */}
     <Form.Item label="资源组" name="resource_id" rules={[{ required: true, message: '请选择资源组' }]}>
       <Select placeholder="请先选择VDI服务器" disabled={!selectedServerId}>
         {resourceGroups.map(group => (
           <Select.Option key={group.resource_group_id} value={group.resource_group_id}>
             {group.name} {group.type ? `(${group.type})` : ''}
           </Select.Option>
         ))}
       </Select>
     </Form.Item>

     {/* Name: auto-generated from resource group name + optional suffix */}
     <Form.Item label="虚拟机名称" name="name" rules={[{ required: true, message: '请输入虚拟机名称' }]}>
       <Input placeholder="选择资源组后自动生成" />
     </Form.Item>
     <Form.Item label="名称后缀（可选）">
       <Input
         placeholder="自定义后缀，例如：user01"
         onChange={(e) => handleSuffixChange(e.target.value)}
       />
     </Form.Item>

     {/* CPU cores - slider */}
     <Form.Item label="CPU 颗数" name="cpu_number" initialValue={1}>
       <Slider min={1} max={16} marks={{ 1: '1', 4: '4', 8: '8', 16: '16' }} />
     </Form.Item>

     {/* CPU cores per socket - slider */}
     <Form.Item label="每颗 CPU 核数" name="cpu_core" initialValue={4}>
       <Slider min={1} max={32} marks={{ 1: '1', 8: '8', 16: '16', 32: '32' }} />
     </Form.Item>

     {/* Memory - slider (GB) */}
     <Form.Item label="内存" name="memory" initialValue={4096}>
       <Slider
         min={512}
         max={65536}
         step={512}
         marks={{ 512: '0.5GB', 4096: '4GB', 8192: '8GB', 16384: '16GB', 32768: '32GB', 65536: '64GB' }}
         tooltip={{ formatter: (v) => v ? `${(v / 1024).toFixed(1)} GB` : '' }}
       />
     </Form.Item>

     {/* Disk - slider (GB) */}
     <Form.Item label="磁盘" name="disk" initialValue={60}>
       <Slider
         min={20}
         max={500}
         step={10}
         marks={{ 20: '20GB', 100: '100GB', 200: '200GB', 500: '500GB' }}
         tooltip={{ formatter: (v) => v ? `${v} GB` : '' }}
       />
     </Form.Item>
   </Form>
   ```

   g. Update the handleCreate function to handle the memory value (currently in MB, slider is in MB already so no conversion needed, but make sure the form value is sent correctly). Remove the line `vdi_server_id: values.vdi_server_id || 'default'` -- the dropdown now provides a real value.

4. Import `VDIResourceGroup` and `VDIServer` types in VirtualMachineList/index.tsx. Also import `Slider` from antd.
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/xingran-go-backend/xingran-react-frontend && npx tsc --noEmit --pretty 2>&1 | head -30</automated>
  </verify>
  <done>
    Create VM modal uses:
    - VDI server dropdown (shows server name, filters by status=0)
    - Resource group dropdown (filtered by selected VDI server, shows group name + type)
    - Name field auto-generated from selected resource group name + optional suffix
    - CPU颗数 slider (1-16 with marks)
    - 每颗CPU核数 slider (1-32 with marks)
    - 内存 slider (0.5-64GB with GB tooltip)
    - 磁盘 slider (20-500GB with GB tooltip)
    TypeScript compilation passes with no errors.
  </done>
</task>

</tasks>

<verification>
1. `go build ./internal/api/v1/vdi/... ./internal/services/vdi/...` passes
2. `cd xingran-react-frontend && npx tsc --noEmit` passes
3. Create VM modal opens with VDI server dropdown populated
4. Selecting a VDI server populates resource group dropdown
5. Selecting a resource group auto-fills the name field
6. Sliders show proper ranges and unit tooltips
</verification>

<success_criteria>
- Backend: New POST /vdi/vm/resource-groups endpoint returns resource groups from local DB
- Frontend: Create VM modal uses Select dropdowns for VDI server and resource group (showing names)
- Frontend: Name auto-generates from selected resource group name with optional custom suffix
- Frontend: CPU cores, memory, and disk use Slider components with unit labels
- TypeScript and Go compilation both pass
</success_criteria>

<output>
After completion, create `.planning/quick/260528-vxy-vdi-id-vdi-id-cpu/260528-vxy-SUMMARY.md`
</output>
