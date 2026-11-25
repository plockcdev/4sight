# Default application database migration design

Use this design for an application's persistent databases unless the application
has a different migration design or its spec explicitly excludes migration
support. Adapt database operations and command names to the application and
backend. Application specs must identify each database, its location and access,
and any deployment-specific requirements. This design is separate from 4sight's
working-database migration bookkeeping.

## Migration chain and build

Maintain one ordered, parent-linked chain per database. Use the SQL entries
prepared for the application's schema under `.4sight/migrations/<dbtype>_<dbname>`
as the source chain. Its initial `00000000-00000000.sql` creates the database;
each subsequent immutable SQL entry transforms its parent schema and data to
the next version. Preserve applied entries and their IDs. Validate unique IDs,
parents, and a single complete chain through the version used by the build;
reject missing, ambiguous, or divergent entries. Do not use filename order or
timestamps as the ordering authority.

At build time embed the complete validated SQL chain through an explicit expected
migration ID in the application binary. The expected ID must be the final
embedded entry required by that binary, not inferred from a working database's
current link. A deployed application needs neither 4sight nor its source files.
Database retirement entries are 4sight project lifecycle operations; do not embed
or execute them as application SQL.

## Applied history and initialization

Keep an application-owned durable migration history for each database, with a
unique migration ID and application timestamp for each successful entry. Specify
its persisted structure and constraints in the schema spec. It is authoritative
for the application's database lifecycle, not a filesystem link or 4sight's
internal tracking table. The initial database creation entry is recorded in this
history once creation and tracking setup succeed. The application's creation
operation may implement that entry using backend-supported operations rather
than running SQL against a nonexistent database, but must have equivalent
results.

Provide an explicit initialization operation for a verified absent or empty
target. It creates the database and history, applies and records the embedded
chain in order through the expected ID, and reports success only when complete.
Do not initialize over existing data or recreate missing tracking state. Treat
partial, damaged, or unrecognized state as an error with recovery guidance.
Ensure initialization is serialized with upgrades and database access. Define
backend-specific identity and emptiness checks before mutation.

## Preflight and upgrade

Before database-dependent application operations, validate the embedded chain
and expected ID and compare the recorded history with it. A complete matching
history permits access. A valid older prefix requires an explicit upgrade; tell
the operator how to run it. A missing, malformed, incomplete, unknown, divergent,
or newer history is an error, not evidence that initialization or a normal
upgrade is safe. Keep help and recovery operations available without this check.
Serialize validation and access with upgrades so that a concurrent upgrade cannot
invalidate the check.

Provide an explicit, repeatable upgrade operation. Under an exclusive lock,
revalidate the target, embedded chain, and recorded history before mutation.
Apply each missing entry in parent order and record it exactly once. Preserve
existing data and include any required data transformations in the migration.
Use atomic migration and history updates when supported. Otherwise make each
interruption recoverable when possible, persist sufficient intent, and stop for
manual repair if actual state cannot be established. Never replay an uncertain
entry or claim success for a partial upgrade. Report the database and last
completed entry on failure. Do not automatically upgrade at ordinary startup.

## Working database coordination

4sight applies the source chain to its working database and records history in
its own bookkeeping. The application's history must agree with that applied
chain before the application uses the working database. During implementation,
reconcile the application history using the verified 4sight history and actual
database state after 4sight applies migrations. Record only entries whose
application is established; never infer success from a submitted link or an
embedded file. Make reconciliation repeatable and safe after interruption. If
either history or the database state is ambiguous or conflicts, stop for repair
without replaying migrations or inventing records. Do not assume 4sight's
bookkeeping is present in deployed environments.

## Verification

Test initialization on a fresh isolated target and upgrade from representative
prior versions with data. Check schema, transformed and preserved data, chain
validation, histories, compatibility, locking, and failure, retry, and repair
behavior. Test coordination with 4sight's working database history separately.
