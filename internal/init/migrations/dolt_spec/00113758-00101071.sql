ALTER TABLE spec.change_review
  ADD COLUMN attempt INT UNSIGNED NOT NULL DEFAULT 1 AFTER change_id,
  DROP FOREIGN KEY fk_change_review_change_id,
  DROP INDEX change_review_change_id_uniq,
  ADD UNIQUE KEY change_review_change_attempt_uniq (change_id, attempt),
  ADD CONSTRAINT fk_change_review_change_id
    FOREIGN KEY (change_id) REFERENCES spec.changes (id);

ALTER TABLE spec.change_review
  ALTER COLUMN attempt DROP DEFAULT;

ALTER TABLE spec.file_change_reviews
  DROP FOREIGN KEY fk_file_change_reviews_file_change_id,
  DROP INDEX file_change_reviews_file_change_id_uniq,
  ADD UNIQUE KEY file_change_reviews_review_file_change_uniq
    (change_review_id, file_change_id),
  ADD CONSTRAINT fk_file_change_reviews_file_change_id
    FOREIGN KEY (file_change_id) REFERENCES spec.file_changes (id);
