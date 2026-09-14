# migrate implement prompt `<path to schema spec relative to schemas dir>`

Output will be a prompt to instruct an agent to read spec and generate a migration

## Prompt Output Specification

Tell agent to read all the design.md and cmd.md docs to inform what the schema needs to provide
(list the files that exist), inferring indexes, foreign keys, primary keys, constraints like
uniqueness, etc.

program will determine the last submitted spec for that db by looking in the migrations dir
for that db, and seeing what submitted points to. This will be known as the parent.

If there was no migrations, then this is the first one.  Prompt the user to add to the
dolt_4sight_migration.md a migration tracking table for the new db like we track them for 4sight_migation then ask it to add the initial migration file to create the (empty) database and create submitted link.

For any migration, tell the agent to generate the migration file for requested
spec for the db:

- show target path for migration calculated from input arg
- show the schema spec path and tell agent to read it

Tell the agent after writing the migration file, to move submitted to the
new migration file

Tell the agent to prompt the user to run `4sight migrate up` when they are ready
to apply changes.
