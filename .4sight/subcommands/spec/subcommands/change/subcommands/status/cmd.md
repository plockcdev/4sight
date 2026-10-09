# 4sight spec change status

Displays the current workspace, submitted, and implemented change IDs.

Read the pointer values from `spec.current_changes`. Render them in a
human-readable report with one entry per pointer:

- Workspace
- Submitted
- Implemented

If the current change pointers are not initialized, report an error.

This command does not modify the database.
