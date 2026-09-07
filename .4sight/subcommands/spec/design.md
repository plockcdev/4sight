# Spec design

We will manage specifications in the spec database.

The "bodies" are the spec files "latest" version, and an agent should be
able to read the bodies and update the application implementation

The can be imported from files on disk

Currently supported are "design.md", "cmd.md" and "coding.md" in the `4sight` directory,
which can have command/subcommand specific files to focus on details related to the
heirarchy.
