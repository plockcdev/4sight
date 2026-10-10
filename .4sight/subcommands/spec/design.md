# Spec design

We will manage code specifications in the spec database.

Currently supported code spec docs, all found in the `.4sight` directory:

- code specs in ./main/ and in subdirs of the various `subcommands` directories
  - design.md - inheritable design across subcommands
  - cmd.md - specific to 1 command
  - coding.md - inheritable coding standards across subcommands
- schema specs only in `.4sight/schemas` directory
  - `<db type>_<db name>.md` - e.g. `dolt_myapp.md`

whenever "design specs" are mentioned, it means the above list

## Bodies

Bodies on disk when compared to db body should always be compared with the
trailing whitespace and empty lines at the end of the document removed before
comparing.

When comparing empty db body, the on disk equivalent is a missing file.

## Resolving bodies and file changes for a change pointer

Whenever resolving a body or file change for a target change pointer, search the
pointer's change ancestry. For each file with a file change on that ancestry,
use only its latest file change on the ancestry and the body associated with
that change.
