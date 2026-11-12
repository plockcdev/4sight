CREATE TABLE spec.filepaths (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  dir_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(255) NOT NULL,
  UNIQUE KEY filepaths_dir_name_uniq (dir_id, name),
  CONSTRAINT fk_filepaths_dir_id FOREIGN KEY (dir_id) REFERENCES spec.dirs (id),
  CONSTRAINT filepaths_name_chk CHECK (
    name REGEXP '^[A-Za-z0-9._-]+$' AND name NOT IN ('.', '..')
  )
);

ALTER TABLE spec.dirs
  DROP CHECK dirs_path_chk;

ALTER TABLE spec.dirs
  ADD CONSTRAINT dirs_path_chk CHECK (
    path = '' OR (
      path REGEXP '^[A-Za-z0-9._/-]+$'
      AND path NOT REGEXP '(^/|/$|//|(^|/)[.][.]?(/|$))'
    )
  );

INSERT INTO spec.filepaths (dir_id, name)
SELECT f.dir_id, f.name
FROM spec.files f;

ALTER TABLE spec.files
  ADD COLUMN current_filepath_id BIGINT UNSIGNED NULL AFTER id;

UPDATE spec.files f
JOIN spec.filepaths fp
  ON fp.dir_id = f.dir_id AND fp.name = f.name
SET f.current_filepath_id = fp.id;

ALTER TABLE spec.file_changes
  ADD COLUMN filepath_id BIGINT UNSIGNED NULL AFTER file_id,
  ADD COLUMN deleted BOOLEAN NOT NULL DEFAULT FALSE AFTER diff,
  ADD COLUMN related_file_id BIGINT UNSIGNED NULL AFTER filepath_id,
  ADD COLUMN similarity DECIMAL(6,5) NULL AFTER related_file_id;

UPDATE spec.file_changes fc
JOIN spec.files f ON f.id = fc.file_id
JOIN spec.filepaths fp
  ON fp.dir_id = f.dir_id AND fp.name = f.name
SET fc.filepath_id = fp.id;

-- Legacy imports represented absent files with an empty result body. A full
-- deletion diff has an empty new side, denoted by +0,0 in its unified hunk.
UPDATE spec.file_changes
SET deleted = (diff LIKE '%+0,0 @@%');

ALTER TABLE spec.bodies
  ADD COLUMN deleted BOOLEAN NOT NULL DEFAULT FALSE AFTER summary;

UPDATE spec.bodies
SET deleted = TRUE
WHERE body = '';

UPDATE spec.files f
JOIN spec.bodies b
  ON b.file_id = f.id AND b.state = 'workspace' AND b.deleted = TRUE
SET f.current_filepath_id = NULL;

ALTER TABLE spec.file_changes
  MODIFY COLUMN filepath_id BIGINT UNSIGNED NOT NULL,
  ADD KEY fk_file_changes_filepath_id (filepath_id),
  ADD KEY fk_file_changes_related_file_id (related_file_id),
  ADD CONSTRAINT fk_file_changes_filepath_id
    FOREIGN KEY (filepath_id) REFERENCES spec.filepaths (id),
  ADD CONSTRAINT fk_file_changes_related_file_id
    FOREIGN KEY (related_file_id) REFERENCES spec.files (id),
  ADD CONSTRAINT file_changes_similarity_chk CHECK (
    similarity IS NULL OR (similarity >= 0 AND similarity <= 1)
  ),
  ADD CONSTRAINT file_changes_related_file_chk CHECK (
    related_file_id IS NULL OR related_file_id != file_id
  );

ALTER TABLE spec.files
  DROP FOREIGN KEY fk_files_dir_id,
  DROP INDEX dir_name,
  DROP CHECK files_name_chk,
  DROP COLUMN dir_id,
  DROP COLUMN name,
  ADD UNIQUE KEY files_current_filepath_id_uniq (current_filepath_id),
  ADD CONSTRAINT fk_files_current_filepath_id
    FOREIGN KEY (current_filepath_id) REFERENCES spec.filepaths (id);
