CREATE TABLE spec.change_id_map (
  old_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
  new_id CHAR(6) NOT NULL,
  UNIQUE KEY change_id_map_new_id_uniq (new_id),
  CONSTRAINT change_id_map_new_id_chk CHECK (regexp_like(new_id, '^ch-[0-9a-z]{3}$'))
);

INSERT INTO spec.change_id_map (old_id, new_id)
SELECT c.id, 'ch-000'
FROM spec.changes c
WHERE NOT EXISTS (
  SELECT 1 FROM spec.changes_closure a
  WHERE a.child_id = c.id AND a.depth > 0
);

INSERT INTO spec.change_id_map (old_id, new_id)
SELECT ranked.id, CONCAT('ch-', LPAD(ranked.n, 3, '0'))
FROM (
  SELECT c.id, ROW_NUMBER() OVER (ORDER BY c.id) AS n
  FROM spec.changes c
  WHERE EXISTS (
    SELECT 1 FROM spec.changes_closure a
    WHERE a.child_id = c.id AND a.depth > 0
  )
) ranked;

ALTER TABLE spec.changes
  ADD COLUMN id_text CHAR(6) NULL AFTER id;

UPDATE spec.changes c
JOIN spec.change_id_map m ON m.old_id = c.id
SET c.id_text = m.new_id;

ALTER TABLE spec.changes
  MODIFY id_text CHAR(6) NOT NULL,
  ADD UNIQUE KEY changes_id_text_uniq (id_text);

ALTER TABLE spec.changes_closure
  ADD COLUMN parent_text CHAR(6) NULL,
  ADD COLUMN child_text CHAR(6) NULL;

UPDATE spec.changes_closure cl
JOIN spec.change_id_map p ON p.old_id = cl.parent_id
JOIN spec.change_id_map c ON c.old_id = cl.child_id
SET cl.parent_text = p.new_id,
    cl.child_text = c.new_id;

ALTER TABLE spec.changes_closure
  MODIFY parent_text CHAR(6) NOT NULL,
  MODIFY child_text CHAR(6) NOT NULL;

ALTER TABLE spec.change_review
  ADD COLUMN change_text CHAR(6) NULL;

UPDATE spec.change_review r
JOIN spec.change_id_map m ON m.old_id = r.change_id
SET r.change_text = m.new_id;

ALTER TABLE spec.change_review
  MODIFY change_text CHAR(6) NOT NULL;

ALTER TABLE spec.current_changes
  ADD COLUMN implemented_text CHAR(6) NULL,
  ADD COLUMN submitted_text CHAR(6) NULL,
  ADD COLUMN workspace_text CHAR(6) NULL;

UPDATE spec.current_changes cur
JOIN spec.change_id_map i ON i.old_id = cur.implemented_change_id
JOIN spec.change_id_map s ON s.old_id = cur.submitted_change_id
JOIN spec.change_id_map w ON w.old_id = cur.workspace_change_id
SET cur.implemented_text = i.new_id,
    cur.submitted_text = s.new_id,
    cur.workspace_text = w.new_id;

ALTER TABLE spec.current_changes
  MODIFY implemented_text CHAR(6) NOT NULL,
  MODIFY submitted_text CHAR(6) NOT NULL,
  MODIFY workspace_text CHAR(6) NOT NULL;

ALTER TABLE spec.file_changes
  ADD COLUMN change_text CHAR(6) NULL AFTER change_id;

UPDATE spec.file_changes fc
JOIN spec.change_id_map m ON m.old_id = fc.change_id
SET fc.change_text = m.new_id;

ALTER TABLE spec.file_changes
  MODIFY change_text CHAR(6) NOT NULL;

ALTER TABLE spec.changes_closure
  DROP FOREIGN KEY fk_changes_closure_parent_id,
  DROP FOREIGN KEY fk_changes_closure_child_id;

ALTER TABLE spec.change_review
  DROP FOREIGN KEY fk_change_review_change_id;

ALTER TABLE spec.current_changes
  DROP FOREIGN KEY fk_current_changes_implemented_change_id,
  DROP FOREIGN KEY fk_current_changes_submitted_change_id,
  DROP FOREIGN KEY fk_current_changes_workspace_change_id;

ALTER TABLE spec.file_changes
  DROP FOREIGN KEY fk_file_changes_change_id;

ALTER TABLE spec.changes
  MODIFY id BIGINT UNSIGNED NOT NULL;

ALTER TABLE spec.changes
  DROP PRIMARY KEY,
  DROP INDEX changes_id_text_uniq;

ALTER TABLE spec.changes
  DROP COLUMN id;

ALTER TABLE spec.changes
  CHANGE COLUMN id_text id CHAR(6) NOT NULL;

ALTER TABLE spec.changes
  ADD PRIMARY KEY (id),
  ADD CONSTRAINT changes_id_chk CHECK (regexp_like(id, '^ch-[0-9a-z]{3}$'));

ALTER TABLE spec.changes_closure
  DROP PRIMARY KEY;

ALTER TABLE spec.changes_closure
  DROP COLUMN parent_id,
  DROP COLUMN child_id;

ALTER TABLE spec.changes_closure
  CHANGE COLUMN parent_text parent_id CHAR(6) NOT NULL,
  CHANGE COLUMN child_text child_id CHAR(6) NOT NULL;

ALTER TABLE spec.changes_closure
  ADD PRIMARY KEY (parent_id, child_id),
  ADD KEY fk_changes_closure_child_id (child_id);

ALTER TABLE spec.change_review
  DROP INDEX change_review_change_id_uniq;

ALTER TABLE spec.change_review
  DROP COLUMN change_id;

ALTER TABLE spec.change_review
  CHANGE COLUMN change_text change_id CHAR(6) NOT NULL;

ALTER TABLE spec.change_review
  ADD UNIQUE KEY change_review_change_id_uniq (change_id);

ALTER TABLE spec.current_changes
  DROP INDEX fk_current_changes_implemented_change_id,
  DROP INDEX fk_current_changes_submitted_change_id,
  DROP INDEX fk_current_changes_workspace_change_id;

ALTER TABLE spec.current_changes
  DROP COLUMN implemented_change_id,
  DROP COLUMN submitted_change_id,
  DROP COLUMN workspace_change_id;

ALTER TABLE spec.current_changes
  CHANGE COLUMN implemented_text implemented_change_id CHAR(6) NOT NULL,
  CHANGE COLUMN submitted_text submitted_change_id CHAR(6) NOT NULL,
  CHANGE COLUMN workspace_text workspace_change_id CHAR(6) NOT NULL;

ALTER TABLE spec.file_changes
  DROP INDEX file_changes_change_file_uniq;

ALTER TABLE spec.file_changes
  DROP COLUMN change_id;

ALTER TABLE spec.file_changes
  CHANGE COLUMN change_text change_id CHAR(6) NOT NULL;

ALTER TABLE spec.file_changes
  ADD UNIQUE KEY file_changes_change_file_uniq (change_id, file_id);

ALTER TABLE spec.changes_closure
  ADD CONSTRAINT fk_changes_closure_parent_id
    FOREIGN KEY (parent_id) REFERENCES spec.changes (id),
  ADD CONSTRAINT fk_changes_closure_child_id
    FOREIGN KEY (child_id) REFERENCES spec.changes (id);

ALTER TABLE spec.change_review
  ADD CONSTRAINT fk_change_review_change_id
    FOREIGN KEY (change_id) REFERENCES spec.changes (id);

ALTER TABLE spec.current_changes
  ADD CONSTRAINT fk_current_changes_implemented_change_id
    FOREIGN KEY (implemented_change_id) REFERENCES spec.changes (id),
  ADD CONSTRAINT fk_current_changes_submitted_change_id
    FOREIGN KEY (submitted_change_id) REFERENCES spec.changes (id),
  ADD CONSTRAINT fk_current_changes_workspace_change_id
    FOREIGN KEY (workspace_change_id) REFERENCES spec.changes (id);

ALTER TABLE spec.file_changes
  ADD CONSTRAINT fk_file_changes_change_id
    FOREIGN KEY (change_id) REFERENCES spec.changes (id);

DROP TABLE spec.change_id_map;
