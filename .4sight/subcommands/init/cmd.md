# 4sight init

Initialize a git repo to have source, schema, and specs managed by 4sight.

Follow the instructions below, implementation must be safely idempotent.

Any failures should indicate the problem and a suggestion on how to solve.

Source paths are relative to the init command directory, target paths are
relative to the git repo root.

## Verify git root

Verify that the current directory is the root of a git repo. If not, then see if
we are in a git repo and instruct on where to run the init command. If not in a
git repo, then give user an example of how to initialize a git repo in a target
directory

## .4sight directory

Create target directory if missing.

### 4sight Build-time migration assets

During implementation of `4sight init`, copy the migrations from the 4sight
source repo's `.4sight/migrations` directory to `internal/init/migrations`, then
embed that directory in the binary.

Copy each database's migration files through the source repo's `current`
pointer. Do not copy migrations after that pointer. Do not use the source repo's
`current` or `submitted` symlink as the target repo's migration state.

## Runtime spec database bootstrap

When a user runs `4sight init`, initialize 4sight's internal spec database in
the target repo's `.4sight/db`:

- Initialize Dolt if `.4sight/db` is not already initialized.
- Apply the embedded `4sight_migration` migration SQL directly from
the binary, in migration order. Track applied migration IDs in the target's
`4sight_migration` database.
- If the tracking table does not exist, apply the initial
`4sight_migration` migration first; it creates the table. Then record that
migration as applied.
- Skip migrations whose IDs are already recorded in the target tracking table.
- Do not invoke `4sight migrate up`. That command manages the target
application's databases.
- Do not create, read, or update migration files or `current`/`submitted`
symlinks in the target repo. This bootstrap applies only to 4sight's internal
`spec` and `4sight_migration` databases.
- After all embedded 4sight_migration migrations have been applied and recorded,
do the same for the spec database.

## Write Embedded Assets

In 4sight code, create symlinks in a `4sight` directory under the init
implementation directory pointing to the files in the assets directory, and then
embed the directory into the 4sight binary.

Leverage the asset contents as instructed, source comes first:

### root_AGENTS.md

search for exact match old (not current) versions of the content and remove,
when comparing, trim each line of begin/end whitespace from both

contents of this asset file should appear only once at the end of the repo root
AGENTS.md, if missing, add it, remove duplicates

track older versions in the embedded directory as copied files for reference,
copying over only when updated in assets. naming is an impmlementation detail

### 4sight_AGENTS.md

overwrite as AGENTS.md in the .4sight directory

### coding.md

if .4sight/coding.md does not exist, create it, otherwise leave alone

## Import Specs

If the workspace change is at ch-000, then init runs `4sight spec change import`
to pick up the initialized spec files
