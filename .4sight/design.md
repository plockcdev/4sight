# 4sight design

## Managed application databases

When an application needs a database and no different database is already
established, default to Dolt. Preserve an established database choice and
project-specific requirements; other database backends have not been tested.
By default use file-backed Dolt without a daemon. Users should commit the Dolt
database files alongside application changes. A Git-based database rollback or
snapshot and restore mechanism is not specified by this design and is deferred;
do not assume that `git reset` safely restores database state.

4sight manages application schema specifications and migration chains. Spec
authoring establishes the initial schema specification, and implementation
prepares and applies initial migrations to bootstrap the working database and
later migrations to evolve it. This workflow does not imply that the application
automatically creates or upgrades databases at runtime or that deployed
instances have been migrated. Specify application database access and any
deployment lifecycle requirements separately when needed.

When existing local data needs to be brought into the established database,
suggest a one-off import tool to run after database setup and before using the
application. This is not an application spec subcommand by default; account for
the project's data and operational needs.

## Command and specification layout

The CLI supports nested subcommands, each of which may have arguments and flags.
Command specifications under `.4sight` follow this structure:

- `main/` describes the command invoked without a subcommand.
- `subcommands/<name>/` describes a named subcommand and may contain its own
  `subcommands/` directory for nested subcommands. A subcommand exists only if
  its directory has a `cmd.md`.
- `schemas/` contains database schema specifications, such as
  `dolt_myapp.md` for the current specification of a Dolt database.

Database schema specifications are the authoritative description of persisted
data: its structure and DDL requirements, meaning and intended use, relationships,
and constraints. Include constraints even when the database cannot enforce them;
the application must enforce any such constraints that apply to its operations.
Schema specs describe the logical model and requirements; SQL DDL is produced in
migration files. Operational choices such as database location, initialization,
and access belong in application design or command specs, unless they are
requirements of the persisted data model itself. See
`.4sight/subcommands/migrate/design.md` for how schema specifications relate to
database migrations.

Code implementation specifications describe the program and its commands. They
may rely on the schema specifications without restating their data definitions,
relationships, or constraints. Reviewers and implementers of code specs must
read and understand the applicable schema specs and ensure the implementation
satisfies them.

At the top level, `design.md` describes the entire program and `coding.md`
contains its coding standards. A command directory may contain `design.md` and
`coding.md`; these are optional. Parent design and coding standards apply along
the command path. A main design or coding standard applies to the main command,
but is not inherited by subcommands unless a spec explicitly says so.

A command's `cmd.md` specifies its arguments, environment variables, flags,
behavior, and output. `design.md` files may describe database usage but must not
contain SQL DDL or statements.

The `main` locator addresses main command specs in spec commands, for example
`4sight spec export main cmd.md`. Discover, track, compare, and include them in
implementation prompts like other code specs.

The shared startup preflight specified in `main/design.md` applies to all
commands as described there.

## Agent Prompts and Agent Responses

By convention, some commands/subcommands may have a nested "agent prompt" and
"agent response" subcommands.

The "agent prompt"" command output provides an agent prompt, which the agent is
to read for information and instructions.

If the agent is to provide information back, it can call various commands, but
the "agent reponse" for the same parent command has some coupling with "agent
prompt"

### Coupling of prompt and response

The "agent reponse" may need more data than comfortably fits in additional
command line arguments. In this case, during an implementation, ensure the
"agent response" implementation has an encoding scheme for everything it needs,
and that the "agent prompt" instructs the agent on the details of the
specification.

The "agent response" command if writing to the db, must receive enough data and
inferable (or concrete) references to satisfy db spec constraints, some of which
are not constrained in schema and need to be constrained by the "agent response"

The "agent prompt" implementation must always instruct agent how to respond
properly.

The actual prompt/response spec is an internal implementation detail and not
part of application specification. If it needs a schema itself (such as a json
schema), then write it to `.4sight/<command path>/agent/responseSchema.<filetype
for schema definition>` so that the implementation of the response can interpret
it and the prompt can provide it to agent. The schema should include as many
constraints specified by the db schema, if can't natively be done in the
response schema then provide as value descriptions.

### Agent Prompt

Some additional rules when interpreting the agent promp spec:

When we say "the agent should XXX" or "instruct agent to XXX" it is meant the
output prompt should be instructing the agent to do XXX".

When we say the user should XX or instruct user to XX, it means the output
prompt should be instructing the agent to instruct the user to XX.

When we say to "stop" it is meant the output prompt should wrap up and we stop
processing, generally an error condition unnless at the end.
