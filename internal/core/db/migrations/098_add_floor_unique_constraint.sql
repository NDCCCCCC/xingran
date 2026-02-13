-- Add unique constraint to ops_floors table on (building_id, floor_no)
-- This ensures that within a building, each floor number is unique

-- First, check and remove any duplicates that might exist
WITH duplicate_floors AS (
    SELECT
        id,
        building_id,
        floor_no,
        ROW_NUMBER() OVER (PARTITION BY building_id, floor_no ORDER BY created_at ASC) as row_num
    FROM ops_floors
    WHERE deleted_at IS NULL
)
DELETE FROM ops_floors
WHERE id IN (
    SELECT id FROM duplicate_floors WHERE row_num > 1
);

-- Now add the unique constraint (excluding soft-deleted records)
CREATE UNIQUE INDEX IF NOT EXISTS idx_ops_floors_building_floor_unique
ON ops_floors (building_id, floor_no)
WHERE deleted_at IS NULL;

-- Record this migration
INSERT INTO schema_migrations (version, description)
VALUES (98, 'add_floor_unique_constraint')
ON CONFLICT (version) DO NOTHING;
