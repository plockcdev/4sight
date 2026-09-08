ALTER TABLE spec.dirs DROP CHECK dirs_chk_bj19leci;

ALTER TABLE spec.dirs
  ADD CONSTRAINT dirs_path_chk CHECK (
    path = '' OR path REGEXP '^[a-z0-9]([a-z0-9.-]*(/[a-z0-9.-]+)*)?$'
  );
