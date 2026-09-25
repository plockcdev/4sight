# 4sight spec change import

Overview: persist filesystem changes as a change

Make a new change a child of the workspace change and update current "workspace" change pointer.  The prompt should be empty,
as a signal that it is an import change.

Create and fill file_changes in db attached to the change for all deltas in spec
between db and workspace by scanning all command dirs for all supported spec filenames
and for all db "workspace" bodies and for each:

- if the "workspace" body does not exist, treat as new spec
- if the workspace body does exist, but local file does not, treat as removed spec
- if db and file differs, the file is the updated version

Print this yaml:

```yaml
- id: new change id
- file_changes:
  - path: path of file change
