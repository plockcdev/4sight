-- Preserve historical review rows. Their original body-summary snapshots were
-- not stored, so mark them explicitly as unavailable. Associate each row with
-- the nearest overall review on the file change's descendant path in the tree.
ALTER TABLE spec.file_change_reviews
  ADD COLUMN change_review_id BIGINT UNSIGNED NULL AFTER file_change_id,
  ADD COLUMN body_summary VARCHAR(65) NULL AFTER summary,
  ADD KEY fk_file_change_reviews_change_review_id (change_review_id);

ALTER TABLE spec.file_change_reviews
  MODIFY COLUMN change_review_id BIGINT UNSIGNED NOT NULL,
  MODIFY COLUMN body_summary VARCHAR(65) NOT NULL,
  ADD CONSTRAINT fk_file_change_reviews_change_review_id
    FOREIGN KEY (change_review_id) REFERENCES spec.change_review (id);
