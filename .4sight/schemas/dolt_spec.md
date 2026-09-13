# Spec Database

The database will store specification data

## bodies

- a string of the body
- fk file_change_id

## changes

Change is

- a record of the original user's change request of the spec for the product
- a reference to a parent change considered this change's baseline
- a name: unique, must start with letter or number,
  must be lowercase, and be only letters, numbers, or hyphen, and limited to 12
  chars.  It can be user or agent provided
- second resolution date + time when the change was created
- agentic provided, which needs to evaluated considering the current state system
  and the change request:
  - why: justification for the change, might be directly provided but also can be
    inferred by agent
  - summary: 72 character or less summary of the change to distinguish it from
    other specs in a list
  - description: paragraph or less summarizing the change if the summary is not
    enough
  - details - full detail of the change if description is not enough

An base change to serve as the root needs to be pre-initialized by migration,
it doesn't specify anything but gives something for other changes to point to

## file_changes

- fk file_id
- fk change_id
- prior_file_change_id: for ordering patches
- diff - a diff patch against the prior version of the file
- created_at

(change_id, file_id, prior_file_change_id) is unique

can only have one null prior_file_change_id for any (change_id, file_id)

file_id must match prior_file_change_id's file_id (when prior_file_change_id is not null)

no prior_file_change_id cycle is allows (nor prior == self)

## files

Files on disk, which for example could be part of a code or schema spec

- fk dir_id
- name: no whitespace file name with combination of lowercase letters,
  hyphens, underscores, numbers, and '.', but not starting with a hyphen or period

## dirs

Filesystem dir path relative to .4sight, could for example be part of a change or a spec

- id
- path: unique directory path, no whitespace with combination of lowercase letters,
  hyphens, numbers, '/', and '.'
  if non-empty, must start with letter or number, can't end with '/' or have two '/' in a row
  empty dir is allowed (root of .4sight)
