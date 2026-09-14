CREATE TABLE spec.changes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  prompt TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO spec.changes (prompt) VALUES ('');

CREATE TABLE spec.changes_closure (
  parent_id BIGINT UNSIGNED NOT NULL,
  child_id BIGINT UNSIGNED NOT NULL,
  depth INT UNSIGNED NOT NULL,
  PRIMARY KEY (parent_id, child_id),
  CONSTRAINT fk_changes_closure_parent_id FOREIGN KEY (parent_id) REFERENCES spec.changes (id),
  CONSTRAINT fk_changes_closure_child_id FOREIGN KEY (child_id) REFERENCES spec.changes (id)
);

INSERT INTO spec.changes_closure (parent_id, child_id, depth) VALUES (1, 1, 0);

CREATE TABLE spec.change_review (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(12) NOT NULL,
  change_id BIGINT UNSIGNED NOT NULL,
  why TEXT NOT NULL,
  summary VARCHAR(65) NOT NULL,
  description TEXT NOT NULL,
  details TEXT NOT NULL,
  score INT UNSIGNED NOT NULL,
  review TEXT NOT NULL,
  UNIQUE KEY change_review_name_uniq (name),
  UNIQUE KEY change_review_change_id_uniq (change_id),
  CONSTRAINT fk_change_review_change_id FOREIGN KEY (change_id) REFERENCES spec.changes (id),
  CONSTRAINT change_review_name_chk CHECK (
    name REGEXP '^[a-z0-9][a-z0-9-]*$' AND CHAR_LENGTH(name) <= 12
  ),
  CONSTRAINT change_review_score_chk CHECK (score <= 100)
);

INSERT INTO spec.change_review (
  name, change_id, why, summary, description, details, score, review
) VALUES (
  'initial',
  1,
  'to provide an anchor for all changes',
  'initialization',
  '',
  '',
  100,
  'perfect'
);

CREATE TABLE spec.current_changes (
  singleton TINYINT NOT NULL DEFAULT 1 PRIMARY KEY,
  implemented_change_id BIGINT UNSIGNED NOT NULL,
  submitted_change_id BIGINT UNSIGNED NOT NULL,
  workspace_change_id BIGINT UNSIGNED NOT NULL,
  CONSTRAINT current_changes_singleton_chk CHECK (singleton = 1),
  CONSTRAINT fk_current_changes_implemented_change_id
    FOREIGN KEY (implemented_change_id) REFERENCES spec.changes (id),
  CONSTRAINT fk_current_changes_submitted_change_id
    FOREIGN KEY (submitted_change_id) REFERENCES spec.changes (id),
  CONSTRAINT fk_current_changes_workspace_change_id
    FOREIGN KEY (workspace_change_id) REFERENCES spec.changes (id)
);

INSERT INTO spec.current_changes (
  implemented_change_id, submitted_change_id, workspace_change_id
) VALUES (1, 1, 1);

ALTER TABLE spec.file_changes
  ADD COLUMN change_id BIGINT UNSIGNED NULL AFTER id;

UPDATE spec.file_changes SET change_id = 1;

ALTER TABLE spec.file_changes
  MODIFY change_id BIGINT UNSIGNED NOT NULL,
  ADD CONSTRAINT fk_file_changes_change_id FOREIGN KEY (change_id) REFERENCES spec.changes (id);

DROP INDEX file_changes_file_prior_uniq ON spec.file_changes;

ALTER TABLE spec.file_changes
  DROP CHECK file_changes_prior_not_self_chk;

ALTER TABLE spec.file_changes
  DROP FOREIGN KEY fk_file_changes_prior_file_change_id;

ALTER TABLE spec.file_changes
  DROP COLUMN prior_file_change_id;

CREATE TABLE spec.bodies_migrated (
  body TEXT NOT NULL,
  file_id BIGINT UNSIGNED NOT NULL,
  file_change_id BIGINT UNSIGNED NOT NULL,
  state VARCHAR(32) NOT NULL,
  ieee_crc32 INT UNSIGNED NOT NULL,
  summary VARCHAR(65) NOT NULL,
  UNIQUE KEY bodies_file_state_uniq (file_id, state),
  CONSTRAINT fk_bodies_migrated_file_id FOREIGN KEY (file_id) REFERENCES spec.files (id),
  CONSTRAINT fk_bodies_migrated_file_change_id FOREIGN KEY (file_change_id) REFERENCES spec.file_changes (id),
  CONSTRAINT bodies_migrated_state_chk CHECK (state IN ('implemented', 'submitted', 'workspace'))
);

INSERT INTO spec.bodies_migrated (body, file_id, file_change_id, state, ieee_crc32, summary)
SELECT
  b.body,
  fc.file_id,
  b.file_change_id,
  'workspace',
  CAST(CRC32(CONCAT(
    SUBSTRING(b.body, 1, 2000),
    SUBSTRING(b.body, 2001, 2000),
    SUBSTRING(b.body, 4001, 2000),
    SUBSTRING(b.body, 6001, 2000),
    SUBSTRING(b.body, 8001, 2000),
    SUBSTRING(b.body, 10001, 2000),
    SUBSTRING(b.body, 12001, 2000),
    SUBSTRING(b.body, 14001, 2000),
    SUBSTRING(b.body, 16001, 2000),
    SUBSTRING(b.body, 18001, 2000),
    SUBSTRING(b.body, 20001, 2000),
    SUBSTRING(b.body, 22001, 2000),
    SUBSTRING(b.body, 24001, 2000),
    SUBSTRING(b.body, 26001, 2000),
    SUBSTRING(b.body, 28001, 2000),
    SUBSTRING(b.body, 30001, 2000),
    SUBSTRING(b.body, 32001, 2000),
    SUBSTRING(b.body, 34001, 2000),
    SUBSTRING(b.body, 36001, 2000),
    SUBSTRING(b.body, 38001, 2000),
    SUBSTRING(b.body, 40001, 2000),
    SUBSTRING(b.body, 42001, 2000),
    SUBSTRING(b.body, 44001, 2000),
    SUBSTRING(b.body, 46001, 2000),
    SUBSTRING(b.body, 48001, 2000)
  )) AS UNSIGNED),
  ''
FROM spec.bodies b
JOIN spec.file_changes fc ON fc.id = b.file_change_id
JOIN (
  SELECT fc2.file_id, MAX(fc2.id) AS max_file_change_id
  FROM spec.bodies b2
  JOIN spec.file_changes fc2 ON fc2.id = b2.file_change_id
  GROUP BY fc2.file_id
) latest ON latest.file_id = fc.file_id AND latest.max_file_change_id = fc.id;

DROP TABLE spec.bodies;

USE spec;

RENAME TABLE bodies_migrated TO bodies;

DELETE fc FROM spec.file_changes fc
JOIN (
  SELECT file_id, MAX(id) AS max_id
  FROM spec.file_changes
  GROUP BY file_id
) latest ON latest.file_id = fc.file_id
WHERE fc.id != latest.max_id;

CREATE UNIQUE INDEX file_changes_change_file_uniq
  ON spec.file_changes (change_id, file_id);

CREATE TABLE spec.file_change_reviews (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  file_change_id BIGINT UNSIGNED NOT NULL,
  why TEXT NOT NULL,
  summary VARCHAR(65) NOT NULL,
  description TEXT NOT NULL,
  details TEXT NOT NULL,
  UNIQUE KEY file_change_reviews_file_change_id_uniq (file_change_id),
  CONSTRAINT fk_file_change_reviews_file_change_id
    FOREIGN KEY (file_change_id) REFERENCES spec.file_changes (id)
);
