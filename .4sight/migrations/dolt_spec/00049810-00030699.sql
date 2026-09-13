ALTER TABLE spec.file_changes
  ADD COLUMN prior_file_change_id BIGINT UNSIGNED NULL AFTER file_id,
  ADD COLUMN created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP;

UPDATE spec.file_changes fc
JOIN (
  SELECT id, LAG(id) OVER (PARTITION BY file_id ORDER BY id) AS prior_id
  FROM spec.file_changes
) chain ON chain.id = fc.id
SET fc.prior_file_change_id = chain.prior_id;

ALTER TABLE spec.file_changes
  ADD CONSTRAINT fk_file_changes_prior_file_change_id
    FOREIGN KEY (prior_file_change_id) REFERENCES spec.file_changes (id),
  ADD CONSTRAINT file_changes_prior_not_self_chk
    CHECK (prior_file_change_id IS NULL OR prior_file_change_id != id);

CREATE UNIQUE INDEX file_changes_file_prior_uniq
  ON spec.file_changes ((
    IF(
      prior_file_change_id IS NULL,
      CONCAT(CAST(file_id AS CHAR), ':0'),
      CONCAT(CAST(file_id AS CHAR), ':', CAST(prior_file_change_id AS CHAR))
    )
  ));
