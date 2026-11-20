# Spec design

We will manage code specifications in the spec database.

Currently supported code spec docs, all found in the `.4sight` directory:

- code specs in ./main/ and in subdirs of the various `subcommands` directories
  - design.md - inheritable design across subcommands
  - cmd.md - specific to 1 command
  - coding.md - inheritable coding standards across subcommands
- schema specs only in `.4sight/schemas` directory
  - `<db type>_<db name>.md` - e.g. `dolt_myapp.md`

whenever "design specs" are mentioned, it means the above list. Text assets
under an `assets` directory owned by a command or subcommand are also tracked
in the shared spec history, but are not design specs.
Binary assets are not tracked in the spec database.

The user-facing spec usage documentation must explain the specification path
conventions in the inherited top-level design, including program-wide, main
command, nested subcommand, and schema spec locations, with examples that help
users find and author the appropriate files for their own application. Use
generic application examples, such as a `build` subcommand or a `myapp`
database, rather than 4sight-specific subcommands or database names.

## Identifying the project during implementation

To decide whether a project is 4sight developing itself, compare the Git
repository root directory name with the repository name from the `origin` fetch
URL. For the remote name, remove a trailing slash, take the final URL or
SCP-style path component, and remove an optional `.git` suffix. Compare names
case-insensitively.

- If both names are `4sight`, identify the project as 4sight developing itself.
- If neither name is `4sight`, identify it as an ordinary application project;
  the two names do not need to match.
- If exactly one name is `4sight`, or either name cannot be determined
  unambiguously, the project identity requires an explicit user answer: 4sight
  itself or an ordinary application.

Do not infer identity from database names or change Git configuration or the
repository directory to affect the result. Names select the implementation
workflow; they do not prove repository authenticity.

## Bodies

Spec bodies on disk when compared to db body should be compared after line
ending normalization, trimming trailing whitespace on each line, and removing
empty lines at the end of the document. Text asset bodies are compared exactly,
including line endings, trailing whitespace, and final newlines.

File absence on disk means deletion; a present empty file is live. The schema
spec for 4sight's spec database defines how this state is stored.

Discover files recursively under command and subcommand `assets` directories,
preserving nested paths. Classify a file as a text asset only if its bytes are
valid UTF-8, contain no NUL, and contain no Unicode control characters except
TAB (U+0009), LF (U+000A), and CR (U+000D). Text asset symlinks are errors.
Treat all other asset files as binary and ignore them during discovery, import,
export, and sync checks.
The stored path model and file identity rules follow that schema
spec. Schema specs are located and named using the conventions above.

Import detects likely renames and copies using line-based similarity. Split the
compared text into lines after applying the file type's comparison rules, then
score a pair as `1 - (added lines + deleted lines) / max(old line count, new
line count)`, with a minimum score of zero. Do not infer relationships between
two empty files. A candidate must score at least 0.60. Automatically associate
a rename only if its score is at least 0.10 higher than every competing
candidate for either path. Otherwise retain delete/add identities and report
the candidates. Detect copies by comparing a new file with all existing files
that remain present, as well as files being renamed. A copy always gets a new
file identity. Record the unique best copy source in `related_file` and its
score when it is at least
0.10 higher than every competing source; otherwise report all qualifying
sources and scores without selecting one. One source may be related to multiple
copies. Similarity and relationship are reviewable metadata, not content
identity.

## Resolving bodies and file changes for a change pointer

Whenever resolving a body or file change for a target change pointer, search the
pointer's change ancestry. For each file with a file change on that ancestry,
use only its latest file change on the ancestry and the body associated with
that change.

## Change pointer lifecycle

The workspace pointer tracks spec edits that can be reviewed and should line up
with the workspace except for unimported specs.

The submitted pointer tracks the latest spec change accepted for implementation.
Code specs, schema specs, and tracked text assets share the same change history
and submission; submit captures their workspace bodies together.

The implemented pointer tracks the latest submitted change whose implementation
is complete.

File identity, path history, deletion, and copy provenance follow the schema
spec for 4sight's spec database; discovery and rename/copy detection follow the
inherited spec design.

