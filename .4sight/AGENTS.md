# 4sight spec driven implementation management of program

## DB Specification

The location for db schema spec lives
in `.4sight/schemas`, these example files are all in the `.4sight/schemas` directory:

`dolt_myapp.md`
  full, current specification for myapp database in a dolt db

## Subcommands

In general the CLI can have nested subcommands, subcommands can have arguments and flags.

Files related to how the commands work and are built can be found according to directories
in .4sight

- main - for the no args main command
- subcommands/* -  each subdir under `subcommands` has a name matching the subcommand name
  and has files related to that named subcommand, and possibly a nested subcommands dir

## Spec Docs

The program design is split across design.md files within the subcommand structure

- `design.md` - design for the entire program
- `main/design.md` - design specific to the main command
- `subcommands/*/design.md` - design files for each subcommand

Each subcommand can have subcommands with their own design docs and subcommands.

The design is inherited from parent design files, so the entire path of design files
have to be read to understand a given subcommand's design.

The design may include db usage, but not SQL DDL or statements.

It is OK to not have a design.md within a given command directory.

There is a cmd.md that is a peer to the design.md, it gives particular specification
for the cli command like input args, env vars, flags, etc. how it behaves, and how
it outputs.  This is code specific.
