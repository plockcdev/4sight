ALTER TABLE spec.files DROP CHECK files_chk_hdm7bg8o;

ALTER TABLE spec.files
  ADD CONSTRAINT files_name_chk CHECK (
    name REGEXP '^[a-z0-9][a-z0-9._-]*$'
  );
