# Main command design

## Internal database startup preflight

Before a command accesses an initialized project's internal databases, run one
shared preflight per process. Help and usage, commands outside a project, and
initial creation by `init` do not require this preflight. An initialized project
missing either internal database or its tracking tables fails with a diagnostic
directing the user to `4sight init` for initialization or recovery.

The binary embeds the complete ordered migration chains and an expected current
migration ID string for each internal database: `4sight_migration` and `spec`.
Compare the actual recorded migration history with those chains. Before any
mutation, validate both embedded chains, expected IDs, and both recorded
histories; each recorded set of IDs must equal the IDs in a complete prefix of
its parent-linked chain. Timestamps do not determine migration order. Fail
on unknown, divergent, incomplete, or newer history. Report the database and
offending migration. Do not infer compatibility by comparing ID strings.

Apply missing embedded migrations in order, first for `4sight_migration`, then
for `spec`, recording each successful migration. Stop on failure, report the
database and migration, and do not run the requested command. Prevent concurrent
preflights from applying migrations to the same project simultaneously. Use
atomic migration and tracking updates where supported; otherwise report that a
failed migration may require repair before retrying. Preflight does not inspect
or migrate application databases.

