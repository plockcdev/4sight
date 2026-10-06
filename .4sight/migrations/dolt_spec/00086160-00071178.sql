-- Preserve historical review rows. Their original body-summary snapshots were
-- not stored, so mark them explicitly as unavailable. Associate each row with
-- the nearest overall review on the file change's descendant path in the tree.
ALTER TABLE spec.file_change_reviews
  ADD COLUMN change_review_id BIGINT UNSIGNED NULL AFTER file_change_id,
  ADD COLUMN body_summary VARCHAR(65) NULL AFTER summary,
  ADD KEY fk_file_change_reviews_change_review_id (change_review_id);

UPDATE spec.file_change_reviews AS fcr
SET change_review_id = (
  SELECT cr.id
  FROM spec.file_changes AS fc
  JOIN spec.changes_closure AS cc
    ON cc.parent_id = fc.change_id
  JOIN spec.change_review AS cr
    ON cr.change_id = cc.child_id
  WHERE fc.id = fcr.file_change_id
  ORDER BY cc.depth ASC, cr.id ASC
  LIMIT 1
);

UPDATE spec.file_change_reviews
SET body_summary = 'unavailable for legacy review';

ALTER TABLE spec.file_change_reviews
  MODIFY COLUMN change_review_id BIGINT UNSIGNED NOT NULL,
  MODIFY COLUMN body_summary VARCHAR(65) NOT NULL,
  ADD CONSTRAINT fk_file_change_reviews_change_review_id
    FOREIGN KEY (change_review_id) REFERENCES spec.change_review (id);
