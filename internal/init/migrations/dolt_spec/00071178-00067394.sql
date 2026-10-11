ALTER TABLE spec.bodies
  ADD PRIMARY KEY (file_id, state);

ALTER TABLE spec.bodies
  DROP INDEX bodies_file_state_uniq;
