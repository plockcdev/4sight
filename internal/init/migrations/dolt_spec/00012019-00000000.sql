CREATE TABLE spec.dirs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  path VARCHAR(255) NOT NULL,
  UNIQUE KEY path (path),
  CHECK (path REGEXP '^[a-z0-9]([a-z0-9.-]*(/[a-z0-9.-]+)*)?$')
);

CREATE TABLE spec.files (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  dir_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(255) NOT NULL,
  UNIQUE KEY dir_name (dir_id, name),
  CONSTRAINT fk_files_dir_id FOREIGN KEY (dir_id) REFERENCES spec.dirs (id),
  CHECK (name REGEXP '^[a-z0-9.-]+$')
);

CREATE TABLE spec.file_changes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  file_id BIGINT UNSIGNED NOT NULL,
  diff TEXT NOT NULL,
  CONSTRAINT fk_file_changes_file_id FOREIGN KEY (file_id) REFERENCES spec.files (id)
);

ALTER TABLE spec.bodies
  ADD COLUMN file_change_id BIGINT UNSIGNED NOT NULL,
  ADD CONSTRAINT fk_bodies_file_change_id FOREIGN KEY (file_change_id) REFERENCES spec.file_changes (id);
