# Spec design

We will manage code specifications in the spec database.

Currently supported code spec docs, all found in the `.4sight` directory:

- code specs in ./main/ and in subdirs of the various `subcommands` directories
  - design.md - inheritable design across subcommands
  - cmd.md - specific to 1 command
  - coding.md - inheritable coding standards across subcommands
- schema specs only in `.4sight/schemas` directory
  - `<db type>_<db name>.md` - e.g. `dolt_myapp.md`

## Bodies

Bodies on disk when compared to db body should always be compared with the
trailing whitespace and empty lines at the end of the document removed before
comparing.

When comparing empty db body, the on disk equivalent is a missing file.

## Finding file change or body for a current change pointer

Since bodies point to file changes and file changes point to changes but don't
get instantiated for every change when the specs are not changed, the body /
file change can be older than the relevant current change pointer. Any time we
are considering bodies / file_changes relative to a current change pointer,
include the ancestry but only use the very latest body/file_change.
