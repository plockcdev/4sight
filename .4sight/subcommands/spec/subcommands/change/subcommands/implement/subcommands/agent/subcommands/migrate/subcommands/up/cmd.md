# 4sight spec change implement agent migrate up

Apply prepared application database migrations for the current submitted spec
change. This command takes no arguments or flags. It is intended for the
implementing agent after code and migration tests pass. It does not prepare
migrations or mark the change implemented. Require a pending submitted change and
matching workspace and submitted specs; stop before mutation if either fails.

Determine the project identity using the inherited spec design. If it requires a
user answer, require the agent to provide that answer on standard input as
exactly one line: `4sight` or `application`. The implementation prompt must tell
the agent to ask the user before supplying the answer. If input is absent,
invalid, or unavailable, stop without mutation and explain that the project
identity must be clarified. Do not save the answer as project configuration.

Run shared preflight and process live submitted application schemas and retired
application databases with a submitted retirement entry. In an ordinary project,
never apply migrations for 4sight's own internal databases; direct the user to
`4sight upgrade` if the running binary requires an internal upgrade.

When 4sight is developing itself, its `4sight_migration` and `spec` databases
are the application databases for this command. The agent invokes the new source
using `go run ./cmd/4sight` from the repository root. For this command only,
shared preflight permits valid recorded histories that are complete older prefixes
of the running binary's embedded chains. It must still reject missing, divergent,
unknown, newer, or otherwise invalid histories before mutation. This exception
does not run `4sight upgrade` or refresh its managed files.

Hold the project lock for the entire operation and revalidate both histories and
target chains under it. Before mutation, verify the submitted source migration
chains, their expected IDs, and their bytes match the running binary's embedded
chains. Apply `4sight_migration` first, then `spec`. Their existing internal
tracking tables are authoritative; never create replacement application tracking
tables for these names. Record each applied ID once and update its source current
link only after successful recording. After migration, use `go run ./cmd/4sight`
for verification and `agent done`. If interrupted or a migration fails, report
the last completed steps and stop for safe reconciliation or repair; never replay
an uncertain migration. Do not claim the change complete on partial success.

Before any application migration, if an application database exists without a
complete trustworthy recorded migration history, stop and require restoration
or explicit reconciliation; never assume it is new after internal database
loss or replay its migration chain.

For each newly introduced application database, create its dedicated migration
tracking table in `4sight_migration` as application migration bookkeeping, using
the schema-defined standard tracking table and safe identifier handling. Create
it only if the application database has never been created and no conflicting
tracking state exists. An interrupted creation may be retried only when durable
evidence identifies it as the same operation and the existing table has the
expected definition and no conflicting history; otherwise fail and report repair
guidance. This bookkeeping
is not an internal version upgrade. Existing application tracking tables and
history must be preserved. Serialize with upgrade and other migration operations.

- consider `.4sight/migrations/<db_type>_<db_name>` as the "migration dir"
- if 00000000-00000000.sql` in migration dir is missing, fail
- if the migration dir is missing a "submitted" symlink to an existing file, fail
- if there is not a path through parent ids from submitted to current, fail
- if at any point in a submitted to current parent traversal, multiple files
  are found with the same parent id, fail
- if the db exists but there is no "current" symlink, fail
- if current and submitted point to the same file, do nothing
- determine the start migration file, call it next migration
  - if the current symlink is present, find migration with it as parent
  - otherwise 00000000-00000000.sql is the starting point
- do a loop until the we have applied the "submitted" migration
  - if the `4sight_migration` table `<db_name>` is missing, fail
  - for an ordinary SQL entry, apply "next migration" and record its id and current timestamp in
    `<db_name>` in the migrations db, atomically where supported
  - after the migration and its tracking record succeed, update current symlink
  - "next migration" is now the migation that has current as parent

If the backend cannot atomically apply and record a migration, make an
interruption recoverable where possible; otherwise stop and report that manual
repair may be needed. Never blindly replay a migration whose application is
uncertain. On retry, validate recorded history and safely reconcile a stale
current symlink to a verified recorded migration before applying more. Report
partial results and enough information for repair, such as the database,
migration ID, and directory. Do not claim success while the symlink and
recorded applied history disagree.


Before normal existence and current-pointer checks, reconcile any incomplete
retirement using the migration design.

For a final retirement entry, first apply all pending preceding SQL migrations
in order. Execute no SQL for the retirement itself; perform the retirement
protocol in the migration design, recording its migration and updating current
only as part of its recoverable completion. A completed retirement is skipped on later
runs, including after garbage collection. Fail on a missing retirement entry for a deleted
submitted schema whose database was created, or a retirement entry that is not final. Never recreate
a retired database from its retained chain.
