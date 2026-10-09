# 4sight spec change submit

Advance the submitted change pointer to the current workspace change.

Before changing any database state, verify that supported spec files on disk
match the workspace bodies, using the same file discovery and trailing
whitespace normalization as `spec change import` and `spec sync-check`. This
includes code and database schema specs. If any file is out of sync, report each
path and the appropriate import or export remediation, then stop without
changing the submitted pointer or submitted bodies.

If the workspace and submitted pointers already match, report that there is no
change to submit and exit successfully.

Otherwise, in one transaction, replace the submitted body cache with copies of
all workspace bodies, preserving each body's file, file change, checksum, and
summary, then set `current_changes.submitted_change_id` to
`current_changes.workspace_change_id`. Files without workspace bodies must
have no submitted body. On failure, leave both the submitted pointer and body
cache unchanged.

After commit, print a success message identifying the submitted change and including its summary from the saved overall change review if there is one.

This command takes no arguments.