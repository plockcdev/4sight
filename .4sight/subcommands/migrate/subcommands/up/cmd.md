# 4sight migrate up

This command will run db migrations so that the databases are up to date with the latest changes.

starting with `4sight_migration` but including all dbs found in the `.4sight/schemas`
directory:

- consider `.4sight/migrations/<db_type>_<db_name>` as the "migration dir"
- if 00000000.sql` in migration dir is missing, fail
- if the migration dir is missing a "submitted" symlink to an existing file, fail
- if there is not a path through parent ids from submitted to current, fail
- if at any point in a submitted to current parent traversal, multiple files
  are found with the same parent id, fail
- if the db exists but there is no "current" symlink, fail
- if current and submitted point to the same file, do nothing
- determine the start migration file, call it next migration
  - if the current symlink is present, find migration with it as parent
  - otherwise 00000000.sql is the starting point
- do a loop until the we have applied the "submitted" migration
  - apply "next migration", update current symlink
  - if the `4sight_migration` table `<db_name>` is missing, fail
  - create row with applied id and current timestamp in `<db_name>`
    in the migrations db
  - "next migration" is now the migation that has current as parent

for failures, give enough information such as directory if relevant
