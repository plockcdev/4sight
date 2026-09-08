# Spec Database

The database will store specification data

## bodies

- a string of the body
- fk file_change_id

## file_changes

- fk file_id
- diff - a diff patch against the prior version of the file

## files

Files on disk, which for example could be part of a code or schema spec

- fk dir_id
- name: no whitespace file name with combination of lowercase letters, hyphens, numbers, and '.'

## dirs

Filesystem dir path relative to .4sight, could for example be part of a change or a spec

- id
- path: unique directory path, no whitespace with combination of lowercase letters,
  hyphens, numbers, '/', and '.'
  must start with letter or number, can't end with '/' or have two '/' in a row
