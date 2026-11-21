Do not treat CHANGELOG as a spec input

# 4sight spec driven implementation management of program

## Specification layout

Always read `.4sight/design.md` before working in the project. It describes the
overall program design.

4sight is intended to manage application schema specifications and migrations.
Unless another database is already established, default to file-backed Dolt
without a daemon. The spec authoring and implementation prompts establish the
schema and bootstrap the working database. Do not assume this automatically
migrates deployed databases or existing local data; discuss a one-off import
tool when needed after database setup and before using the application.
Users should commit the Dolt database files alongside application changes.
Do not assume `git reset` safely restores database state; a Git-based database
rollback mechanism is deferred.

Database schema specifications live in `.4sight/schemas`. Other specifications
describe the program and its commands:

- `.4sight/design.md` and `.4sight/coding.md` contain the overall design and
  coding standards.
- `.4sight/main/` contains specifications for the command invoked without a
  subcommand.
- `.4sight/subcommands/<name>/` contains specifications for a subcommand and may
  itself contain a `subcommands/` directory for nested subcommands. A subcommand
  exists only if its directory contains `cmd.md`.
- A command's `cmd.md` specifies its arguments, environment variables, flags,
  behavior, and output. Its optional `design.md` and `coding.md` describe
  command-specific design and standards.

Read the applicable parent design and coding files along a command's path. Design
files may describe database usage but must not contain SQL DDL or statements.
See the migrate design for how database schema specifications are used.

# Implementing Specs

When asked to implement the spec, including an explicitly code-only or
schema-only request, run `4sight spec implement prompt` and follow its
instructions for the requested scope.

Do not change or import code or schema specs during implementation.

# Handling User request for spec review

Before a requested spec review, check for un-imported spec changes. If any
exist, stop and tell the user to import them or reset the workspace specs.

When the user requests a spec review, run `4sight spec change review agent
prompt` and follow the instructions.

If the current workspace change already has a review and the user asks for a
review, stop and tell them a review already exists. Ask them to request a
re-review explicitly. When they request a re-review, run `4sight spec change
review agent prompt` and follow the instructions for a new numbered attempt.

After saving a review, show the user the change ID, review attempt, readiness,
and the full change-level review text. If the change is not ready, also present
all actionable blockers and unresolved decisions from the overall and
file-level findings, with enough context for the user to respond. Include the
command `4sight spec change review show <change-id> <attempt>` for the complete
saved review. Apply this guidance to reviews following spec change requests too.

# Handling spec change requests

Before handling a spec change request, check for un-imported spec changes. If
any exist, stop and tell the user to import them or reset the workspace specs.

When the user requests a spec change, including a code-spec change, a
schema-spec change, or both, run `4sight spec change request <prompt>` with their
request. Then run `4sight spec change author agent prompt` and follow its
instructions. A change request that affects both code and schema specs should
include both in the same authoring workflow.
