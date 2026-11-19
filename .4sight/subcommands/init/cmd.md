# 4sight init

Initialize a git repo to have source, schema, and specs managed by 4sight.

Bootstrap when the internal database location is absent or empty, even when
managed project files already exist. File setup must be safely idempotent.

Any failures should indicate the problem and a suggestion on how to solve.

Source paths are relative to the init command directory, target paths are
relative to the git repo root.

## Verify git root

Verify that the current directory is the root of a git repo. If not, then see if
we are in a git repo and instruct on where to run the init command. If not in a
git repo, then give user an example of how to initialize a git repo in a target
directory

## Preserve workspace changes

Do not require a clean working tree. Preserve unrelated files and changes; do
not stage or commit files as part of initialization.

## .4sight directory

Create target directory if missing.

## .gitignore

Add `.4sight/changes` to the target repository root `.gitignore` if it is not
already present. Preserve existing entries and do not add duplicates.

### 4sight Build-time migration assets

During implementation of `4sight init`, copy the migrations from the 4sight
source repo's `.4sight/migrations` directory to `internal/init/migrations`, then
embed that directory in the binary.

For each internal database, select the final migration ID that the binary's
implemented code requires. Copy the complete validated parent-linked source
chain through that ID into the embedded assets, including newly authored
migrations even if the source `current` symlink is still behind. The expected ID
must match the last embedded migration. Do not infer the intended build version
from a target project's symlinks or embed later migrations not implemented by this
binary. Internal migration assets are authoritative at runtime.

## Runtime spec database bootstrap

Before changing files or databases, inspect `.4sight/db`. If it is absent or
empty, bootstrap it, regardless of existing managed files. If it contains a
complete initialized installation, stop and direct the user to `4sight upgrade`.
If it is nonempty but partial, damaged, or unrecognized, stop without changing
it and report the location and recovery guidance. Never overwrite or recreate a
partially present internal database or tracking table.

Bootstrapping after database loss cannot recover prior spec history, reviews,
change pointers, application migration histories, or retirement records. Report
this limitation. Never infer application migration state from a fresh internal
database or replay migrations on an existing application database; require
restoration or explicit reconciliation of its history before migration.

When a user runs `4sight init`, initialize 4sight's internal spec database in
the target repo's `.4sight/db`:

- Initialize Dolt if `.4sight/db` is not already initialized.
- Apply the embedded `4sight_migration` migration SQL directly from
the binary, in migration order. Track applied migration IDs in the target's
`4sight_migration` database.
- If the tracking table does not exist, apply the initial
`4sight_migration` migration first; it creates the table. Then record that
migration as applied.
- Record every successfully applied migration; if initialization is interrupted,
  report recovery instructions instead of treating a subsequent init as a fresh project.
- Do not invoke `4sight migrate up`. That command manages the target
application's databases.
- Do not write internal migration files or create internal migration symlinks in
the target repo. If existing internal current symlinks and source files are
present, reconcile them to the newly recorded state as specified for upgrade. This bootstrap applies only to 4sight's internal
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

After bootstrapping a new database, import any existing supported specification
files and tracked text assets from disk using `4sight spec change import`, so
the workspace database reflects them. This recreates the current workspace from
the files; previous database history cannot be recovered by the import. 

