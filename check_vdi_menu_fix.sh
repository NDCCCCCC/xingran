# 这个脚本生成正确的 SQL 来分配 VDI 菜单权限

echo "-- 检查 sys_role 表结构"
echo "SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'sys_role' ORDER BY ordinal_position;"
echo ""
echo "-- 查看管理员角色"
echo "SELECT * FROM sys_role WHERE role_key = 'admin' LIMIT 1;"
echo ""
echo "-- 查看 sys_role_menu 表结构"  
echo "SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'sys_role_menu' ORDER BY ordinal_position;"
