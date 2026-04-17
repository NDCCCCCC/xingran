-- Increase device_code column size from 50 to 200 to accommodate longer device codes
ALTER TABLE ops_room_devices ALTER COLUMN device_code TYPE varchar(200);
