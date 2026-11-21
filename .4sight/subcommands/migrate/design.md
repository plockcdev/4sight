# migrations

Migrations are in charge of the schema state of the database by applying
sql files in sequence to align the db with the current implemented state of the code.

The schema spec can determine a single SQL script that can define the schema
from scratch.

However, the final perfect SQL is not known in advance.  There is an initial
schema applied to an empty database for the initial set of features, then a
series of sql changes to the schema as the program
matures and feature are added or taken away.  These changes are driven by
schema spec changes.  To update the db from empty to each successive version
of the schema, we call these "migrations".

## file layout

### migrations

Ordinary migration files are SQL scripts. A retirement migration is a separate
`.delete` chain entry with no SQL file, as specified below.

New migrations that implement a live submitted schema include these SQL line
comments at the beginning of the file, once each:

```text
-- 4sight-schema: <path relative to .4sight/schemas>
-- 4sight-schema-change: <submitted change ID>
-- 4sight-schema-crc32: <eight lowercase hexadecimal digits>
-- 4sight-schema-sha256: <64 lowercase hexadecimal digits>
```

The path and change ID identify the submitted schema body the migration is
intended to result in; resolve its stable file identity through spec history.
Compute IEEE CRC32 and SHA-256 over the UTF-8 bytes of that body after the spec
comparison normalization in spec design. Normalize line endings to LF, trim
trailing whitespace on each line, remove trailing empty lines, and end a
nonempty normalized body with one LF. An empty body has zero bytes. The CRC32
must agree with the submitted body's stored `ieee_crc32` when that checksum
was calculated from the normalized body; otherwise calculate it from the body.
SHA-256 is recorded as metadata and is not used for matching.

The initial migration that only creates an empty database and a retirement
migration do not represent a live resulting schema and must not claim these
checksums. Existing migrations without this metadata remain valid migration
chain members; their absent metadata cannot establish a schema match.

These sql migration files will live in `.4sight/migrations/` with `.sql` extension.

Each migration, including a retirement, has an id.

The id is a string of 8 numbers, the first 4 are incrementing based on the last
"submitted" id's first 4 digits but incremented by 1 (to give an easy to read
ordering) and the last 4 are a random number to help with collisions.  The id is
used for 4sight_migrations table ids and also for file names

On disk the filename has the id and also a parent to establish a sequence of patches.

Files in that dir, db_type example is "dolt", db_name example is "myapp":

- `<dbtype>_<dbname>/<id>-<parent_id>.sql`
- `<dbtype>_<dbname>/<id>-<parent_id>.delete` (retirement only)
- `<dbtype>_<dbname>/current` -> the applied `.sql` or `.delete` entry
- `<dbtype>_<dbname>/submitted` -> the submitted `.sql` or `.delete` entry

current symlink is based on most recent query of "new" for the most recent migration in the db.  if the table does not exist, there is no current symlink and we consider it uninitialized

There can be many SQL entries in a chain, followed by at most one final
retirement entry. IDs and parent links must be unambiguous across both types.

the first migration is always 00000000-00000000.sql and creates the database.

## database migration state tracking

We will have a migrations db called "4sight_migration", this is known as
the "migrations" db, and it has migration tables, one per database, named
`<db_name>` (no db type prefix).
The schema defines the standard application tracking table. `spec change implement agent migrate up` creates
it as application bookkeeping before the first migration of a new application
database, subject to its safety checks. Internal migration history is managed by `init` and `upgrade` using the
binary's embedded chains and the internal tracking tables. When 4sight develops
itself, the agent migration command applies its own prepared source chains as
application migrations, using those same tracking tables as specified by that
command. Existing internal current symlinks are optional representations, never
authoritative. Upgrade leaves them untouched in ordinary application projects;
in the 4sight source repository, reconcile them as specified by upgrade and
the self-hosted agent migration command.
Runtime commands never write internal migration SQL files to projects.


## Removed databases

A submitted deletion of a managed application schema retires its database if
it was created; a
missing unsubmitted file alone does not. Compare implemented and submitted
schema identities and paths, including deleted bodies, rather than discovering
only existing files. A schema rename that changes database identity retires the
old database and introduces the new one. The internal `4sight_migration` and
`spec` databases cannot be retired through this mechanism; report a spec defect.

Represent retirement as a final migration in the old database's existing chain.
It is represented by a standalone `<id>-<parent_id>.delete` entry, with no
corresponding SQL file or user-authored SQL. The entry participates in the same
ID, parent, submitted, and current chain rules as ordinary migrations. Reject
entries with duplicate IDs, an SQL file for the same entry, or children after
retirement. Apply all pending preceding migrations in order before retirement;
any required schema or data changes must be ordinary preceding migrations.
Keep the chain and its tracking table for audit and recovery. A retirement is
identified by database type, original name, and migration ID. A later database
with the same name must not reuse a retired chain or its recorded history;
stop and report the conflict until a distinct lifecycle is supported.

When applying retirement, move the database to a unique physical name of the
form `trashed_db_<unique id>_<original name>`, subject to backend name limits,
without overwriting any existing database. The original name must cease to
resolve to the retired database for applications, while its contents remain
available under the trash name. Record the retirement in the migrations db.
Use backend-supported operations that preserve database contents; if unavailable,
fail without destroying the database. Serialize retirement and garbage
collection for the project. Make the migration, physical move, tracking record,
and current pointer recoverable across failure or interruption: retries must
recognize the same operation, never trash an unrelated database, and never
create duplicate records. If atomicity is unavailable, persist sufficient
intent before the move, reconcile each intermediate state on retry, and report
any state that cannot be safely reconciled for manual repair. Persist the
retirement intent and chosen trash name before moving. Record the retirement
migration ID in the application tracking table only after the move is verified;
then update current to the `.delete` entry. Persist enough phase information to
resume each step idempotently. On retry, verify the tracked operation and physical
identity before recording a missing migration ID or repairing a stale current
link; never repeat a completed move. If the record, identity, or pointer cannot
be reconciled safely, stop for manual repair. For an incomplete move,
if only the original exists, perform the move; if only the trash name exists,
finish recording it. If both exist or both are absent, stop and report the
unexpected state. Never claim a
retirement complete until all its effects are recorded.
