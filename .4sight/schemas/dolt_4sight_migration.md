# 4sight migration

There needs to exist a db called "4sight_migration"

The `4sight_migration` and `spec` tables track the applied migrations of the
corresponding internal databases. Each has a user-provided migration ID as its
primary key and a required second-resolution application timestamp. Their
records are authoritative for internal migration state and must correspond to a
complete prefix of the respective embedded migration chain.


## Application migration tracking tables

Each managed application database has a dedicated table named for that database,
containing unique user-provided migration IDs and a required second-resolution
applied timestamp. These tables are application migration bookkeeping and are
created when the application database is first introduced. Their names must not
collide with internal tracking tables or other managed database names. Preserve
them and their history when an application database is retired.


## database_deletions

An audit record for each retirement of a database that was created:

- database type and original database name, both required
- trash database name, required and unique across records
- deleting migration ID, required; together with database type and original
  name uniquely identifies the retirement and refers semantically to that
  database's migration chain
- trashed_at: date and time of successful move to trash; null until completed
- garbage_collected_at: date and time of successful permanent deletion; null
  until collected
- state and any durable operation information needed to recover an interrupted
  retirement or garbage collection

Keep records after collection. A collected record must have a trash date and a
collection date no earlier than the trash date. A database cannot be collected
before its retirement is complete. The application enforces cross-database and
lifecycle constraints that cannot be enforced by the database.
