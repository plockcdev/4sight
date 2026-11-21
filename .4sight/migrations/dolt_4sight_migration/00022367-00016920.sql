-- 4sight-schema: dolt_4sight_migration.md
-- 4sight-schema-change: ch-178
-- 4sight-schema-crc32: 6f29e089
-- 4sight-schema-sha256: fced8e3d3a8f22d42f099f3431cb3efef137c1ac2212c0c67bc54533975ed8c1
CREATE TABLE 4sight_migration.database_deletions (
  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  database_type VARCHAR(32) NOT NULL,
  original_name VARCHAR(64) NOT NULL,
  trash_name VARCHAR(64) NOT NULL,
  migration_id CHAR(8) NOT NULL,
  trashed_at DATETIME NULL,
  garbage_collected_at DATETIME NULL,
  state VARCHAR(32) NOT NULL,
  physical_device BIGINT UNSIGNED NOT NULL,
  physical_inode BIGINT UNSIGNED NOT NULL,
  UNIQUE KEY (trash_name),
  UNIQUE KEY (database_type, original_name, migration_id)
);
