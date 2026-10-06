# 4sight spec change request `<prompt>`

Persist a user's prompt as the updated current workspace change

If no prompt, do not persist anything and tell user to provide a prompt.

Make the new change a child of the workspace change and update current change
pointer for the workspace

Print this yaml:

```yaml
id: new change id
```
