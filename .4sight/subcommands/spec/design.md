# Spec design

We will manage code specifications in the spec database.

Currently supported code spec docs, all found in the `.4sight` directory:

- code specs in ./main/ and in subdirs of the various `subcommands` directories
  - design.md - inheritable design across subcommands
  - cmd.md - specific to 1 command
  - coding.md - inheritable coding standards across subcommands
- schema specs only in `.4sight/schemas` directory
  - `<db type>_<db name>.md` - e.g. `dolt_myapp.md`
