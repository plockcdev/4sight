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

### Specs

The location for db schema spec lives
in `.4sight/schemas`, these example files are all in the `.4sight/schemas` directory:

`dolt_myapp.md`
  full, current specification for myapp database in a dolt db

### migrations

Each migration file is a DDL (sql) script.

These sql migration files will live in `.4sight/migrations/` with `.sql` extension.

Each SQL file mutation of the database has an id.

The id is a string of 8 numbers, the first 4 are incrementing based on the last
"submitted" id's first 4 digits but incremented by 1 (to give an easy to read
ordering) and the last 4 are a random number to help with collisions.  The id is
used for 4sight_migrations table ids and also for file names

On disk the filename has the id and also a parent to establish a sequence of patches.

Files in that dir, db_type example is "dolt", db_name example is "myapp":

- `<dbtype>_<dbname>/<id>-<parent_id>.sql`
- `<dbtype>_<dbname>/current` -> `<id>-<parent_id>.sql`
- `<dbtype>_<dbname>/submitted` -> `<id>_<parent_id>.sql`

current symlink is based on most recent query of "new" for the most recent migration in the db.  if the table does not exist, there is no current symlink and we consider it uninitialized

there can be many `<id>-<parent_id>.sql` files to define a chain

## database migration state tracking

We will have a migrations db called "4sight_migration", this is known as
the "migrations" db, and it has migration tables, one per database, named
`<db_name>` (no db type prefix).
