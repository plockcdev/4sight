# 4sight spec change import

Overview: persist filesystem changes as a change

If differences are found, make a new change a child of the current workspace
change and update the current "workspace" change pointer to it. The prompt
should be empty, as a signal that it is an import change. If no differences are
found, leave the workspace pointer unchanged, print "no changes found" to
stderr, and exit cleanly.

Create and fill file_changes for all deltas between the on-disk supported spec
files and text assets and the workspace database state. Follow the discovery,
comparison, and rename/copy rules in the inherited spec design, and the file,
filepath, and deletion model in the schema spec.

For every detected difference, write the corresponding file change and
workspace body state according to the schema, replacing any prior workspace
body for that file. Preserve content according to the inherited comparison
rules.

Print this yaml:

```yaml
id: new change id
file_changes:
  - path: path of file change relative to .4sight
    summary: the +/- count of lines in the diff
```
