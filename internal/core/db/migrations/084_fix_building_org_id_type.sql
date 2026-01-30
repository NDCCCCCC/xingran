-- Fix building org_id type mismatch
-- This migration fixes buildings where org_id contains department names instead of UUIDs
-- Issue: Some buildings have org_id set to department names (e.g., "巴东县野三关镇营销服务部")
-- Expected: org_id should be a valid UUID from sys_dept table

-- Step 1: First, let's see what we're fixing (as a comment)
-- The issue occurs when org_id is NOT a valid UUID but matches a department name

-- Step 2: Update buildings where org_id is a department name
-- We join with sys_dept on dept_name to get the correct UUID
UPDATE ops_buildings
SET org_id = sys_dept.id::text,  -- Convert UUID to text for varchar column
    -- Also update org_name for consistency
    org_name = sys_dept.dept_name,
    updated_at = NOW()
FROM sys_dept
WHERE ops_buildings.org_id IS NOT NULL
  -- Check if org_id is NOT a valid UUID format (basic check: length != 36 or doesn't contain '-')
  AND (LENGTH(ops_buildings.org_id) != 36 OR ops_buildings.org_id NOT LIKE '%-%')
  -- Match department name
  AND ops_buildings.org_id = sys_dept.dept_name;

-- Step 3: For any remaining buildings with invalid org_id, set to NULL if no match found
-- This handles cases where the department name doesn't exist anymore
UPDATE ops_buildings
SET org_id = NULL,
    org_name = NULL,
    updated_at = NOW()
WHERE org_id IS NOT NULL
  AND (LENGTH(org_id) != 36 OR org_id NOT LIKE '%-%')
  AND NOT EXISTS (
    SELECT 1 FROM sys_dept WHERE sys_dept.id::text = ops_buildings.org_id::text
  );

-- Step 4: Add a comment to document the fix
COMMENT ON COLUMN ops_buildings.org_id IS '所属机构ID（关联sys_dept.id，必须是有效的UUID）';

-- Step 5: Optional - Add a check constraint to prevent future issues (PostgreSQL)
-- Note: This may fail if there are still invalid entries, but step 2-3 should have fixed them
ALTER TABLE ops_buildings
ADD CONSTRAINT chk_building_org_id_is_uuid
CHECK (org_id IS NULL OR org_id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$');

-- Step 6: Create index for better query performance
CREATE INDEX IF NOT EXISTS idx_ops_buildings_org_id
ON ops_buildings(org_id) WHERE org_id IS NOT NULL;
