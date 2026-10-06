-- Convert the numeric review score to the boolean readiness field defined by
-- the schema. Existing scores above 50 are ready; scores of 50 or less are
-- not ready, except ch-011 which is explicitly ready.
UPDATE spec.change_review
SET score = CASE
  WHEN score > 50 OR change_id = 'ch-011' THEN 1
  ELSE 0
END;

ALTER TABLE spec.change_review
  DROP CHECK change_review_score_chk;

ALTER TABLE spec.change_review
  CHANGE COLUMN score ready BOOLEAN NOT NULL;

ALTER TABLE spec.change_review
  ADD CONSTRAINT change_review_ready_chk CHECK (ready IN (0, 1));
