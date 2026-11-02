# 任务状态：进行中

## 已完成
✅ 删除前端 group-mapping 页面目录
✅ Git提交保存进度 (commit: 229238f)

## 待完成（下个session继续）
- [ ] 从 adDomainApi.ts 移除部门-组映射API函数
- [ ] 从路由配置删除 group-mapping 路由引用
- [ ] 删除后端部门-组映射相关文件：
  - internal/api/v1/system/ad_group_mapping_router.go
  - internal/services/addomain/dept_group_mapping_service.go
  - internal/models/dept_group_mapping.go（标记废弃）
- [ ] 创建 OU-组映射模型和服务
- [ ] 创建 OU-组映射 API
- [ ] 更新 OU 页面UI：
  - 删除"关联部门信息"卡片
  - 添加"关联用户组"管理功能
- [ ] 编译验证

## 备注
任务因context限制暂停。下个session从"待完成"列表继续执行。
