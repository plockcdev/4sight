# Spec design

We will manage code specifications in the spec database.

The "bodies" are the spec files "latest" version, and an agent should be
able to read the bodies and update the application implementation

Specs can be imported from files on disk

Specs can be exported to disk.

Currently supported code spec docs, all found in the `.4sight` directory:

- design.md - inheritable design across subcommands
- cmd.md - specific to 1 command
- coding.md - inheritable coding standards across subcommands
