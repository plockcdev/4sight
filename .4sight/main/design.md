# Main command design

## Internal database startup preflight

For every command in an initialized project, run one shared read-only
preflight per process. Help and usage remain available without
preflight, including for uninitialized, outdated, or damaged projects. Initial
creation by `init` uses its own checks. `upgrade` uses the same validation but
is allowed to proceed when valid histories are behind; all other database
commands, including `migrate up`, stop before mutation in that case and tell the
user to run `4sight upgrade`.

The binary embeds the complete ordered migration chains and an expected current
migration ID for each internal database: `4sight_migration` and `spec`. Validate
both chains and expected IDs and compare the recorded histories in
`4sight_migration.4sight_migration` and `4sight_migration.spec` with their
chains. These records, not filesystem symlinks, are authoritative for applied
internal migration state. Each recorded history must be a complete prefix of its parent-linked
chain; timestamps and lexicographic ID order do not determine migration order.
Validate both before reporting upgrade required. Unknown, divergent, incomplete,
newer, malformed or missing history, databases or tracking tables are errors:
identify the database and offending state, do not tell the user an upgrade alone
will repair it, and give safe recovery guidance. Never mutate during preflight.
If both histories are valid but at least one is behind, report upgrade required;
if both are current, allow the command to proceed.

Serialize validation and database access as needed with upgrade and application
migration so that a concurrent upgrade cannot invalidate a successful check.
Preflight does not inspect or migrate application databases.
