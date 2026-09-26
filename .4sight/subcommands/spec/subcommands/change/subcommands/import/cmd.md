# 4sight spec change import

Overview: persist filesystem changes as a change

When needed, make a new change a child of the workspace change and update current "workspace" change pointer.  The prompt should be empty,
as a signal that it is an import change.
If there turns out not to be any file changes (see below), print to
stderr no changes found, and exit cleanly

Create and fill file_changes in db attached to the change for all deltas in spec
between db and workspace by scanning all command dirs and schema dir for all supported spec filenames
and for all db "workspace" bodies and for each:

- if the "workspace" body does not exist, treat as new spec
- if the workspace body does exist, but local file does not, the spec diff is removing
  the entire contents of the spec
- if db and file differs, the file is the updated version
- create the "workspace" body for the file, without summary, and remove
  the prior "workspace" body if it existed

Note that anytime handling the filesystem file (comparing, diffing, writing to body),
ignore trailing whitespace and empty lines at the end of the file.

Print this yaml:

```yaml
id: new change id
file_changes:
  - path: path of file change relative to .4sight
```
