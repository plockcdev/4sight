ALTER TABLE spec.change_review
  DROP CHECK change_review_score_chk;

ALTER TABLE spec.change_review
  CHANGE COLUMN score ready BOOLEAN NOT NULL;

ALTER TABLE spec.change_review
  ADD CONSTRAINT change_review_ready_chk CHECK (ready IN (0, 1));
