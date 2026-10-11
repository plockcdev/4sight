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

## Runtime DB Setup

When a user runs `4sight init` in a target repo:

1. Initialize Dolt in the target repo's `.4sight/db` if it is not initialized.
2. Apply the embedded `4sight_migration` migrations in order, using the target
database's tracking table to skip migrations already applied. 3. Apply the
embedded `spec` migrations in order using the same target tracking state. 4.
Record each applied migration in the target's migration tracking database.

Do not modify the target repo's migration files or symlinks as part of this
embedded bootstrap.

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
