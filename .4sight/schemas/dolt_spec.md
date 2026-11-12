# Spec Database

The database will store specification data

Commands that insert or update any row in the spec database must satisfy every
constraint and system-maintained column defined in this document, Individual
command specifications do not restate these rules, obeying them is implicit.

## bodies

A body is

- a full text of the spec body acting as a cache
it should always be set but can be empty, such as when pointing at the root
change or if the latest file change empties the body
- file: reference to which file this body is for
- file change: reference to which file change was the end of the change sequence
for this file, its file reference must match ours
- state: one of the kinds of "current changes", implemented, submitted, workspace
- ieee_crc32: checksum of the body, should always be set
- summary: 65 character or less summary of what the body text specifies
- deleted: whether this state represents a deleted file; an empty body with
  deleted false is a present empty file

The body is a full text of the file change for its file (one body per file per
state, which should be the primary key)

The state plus file reference can infer a sequence of file changes.

Applying the diffs from the file changes in order of the change sequence will
give the full text

No body exists for a file that has never been changed for the given state.

the referenced file change must be the last file change for this file on the
change sequence for this state's change, note that the last file change can be
on an earlier change than the state referenced change if no following changes to
that file were made

## changes

Change is

- an identifier in the form of `ch-001` where the last goes up to 999, then
the increment the most significant char from a-z until reaching zzz. Example
001, 002, ...998, 999, a00, a01, ... b00, ...z99, za0, za1, ...zzz It is
globally unique
- a prompt which is the prompt text recording the original user's change request of
the spec for the product
- second resolution date + time when the change was created

Additionally, we need to store reference to changes in a tree structure backed
by a closure table we can call changes_closure (parent/child/depth)

- There is only one tree for all changes.
- A base change serves as the root change and should be
pre-initialized by migration, the prompt for root node is empty and needs to be
seeded in changes_closure

A change sequence is the root node path through the tree to any change node.

## change_review

Captures an agentic review of a change

- change: reference to which change this review is for, only one change_review
per change
- why: justification for the change, might be directly provided but also can be
inferred by agent, must be non-empty
- summary: 65 character or less summary of the change to distinguish it from
other specs in a list, must be non-empty
- description: paragraph or less summarizing the change if the summary is not
enough
- details - full detail of the change if description is not enough
- review: review of the spec change for quality of specification and logic consistency, must be non-empty
- ready: true/false if spec meets minimum specification to implement the change

## current_changes

- implemented: the latest implemented change, initialized to the root change
- submitted: the latest submitted change, initialized to the root change
- workspace: the change seen in the workspace, initialized to the root change

This will be initialized by migration, after which:

- there must not be more than one set of current changes
- the change references must always be set
- workspace can be the same as submitted or a descendant of submitted
- submitted can be the same as implemented or a descendant of implemented

## file_changes

- represent an incremental change of a referenced file
- change: reference to the change this file change is part of
- file: stable identity of the file this diff is calculated for
- diff - what changed in unified diff format
- deleted - whether this change makes the file absent; false means a present
  file, including a present empty file
- filepath: reference to the filepath where this file exists after this change;
  for deletion, retain the filepath of the last existing path
- related_file: optional source file identity for copy provenance only; copies
  have their own file identity
- similarity: optional line-based score for an inferred rename or copy
- created_at: second resolution date and time

File changes attach to a stable file identity, not to a path. A file identity
persists through a rename. Its filepath reference on each file change records
the path at that point in history; the previous file change on the ancestry
provides the old path. Deleting a file releases its current path; creating a
file at a deleted path creates a new identity and reuses the filepath row.
Current paths must be unique among live files. Rename similarity is recorded
without a related-file reference because the same file identity continues.
Copy similarity is recorded with the source in related_file. Similarity is
stored for selected rename/copy relationships; ambiguous copy candidates are
reported without storing a selected source.

There can be only one file change per file per change, but a change can have
multiple file changes by pointing to different files

Conceptually, a file change is created based on the current workspace's parent
change (as the new workspace change must be created before any file changes),
which should match the current body text for the file as the file change has to
come before the body update.

If no previous file change found, baseline is empty string.

## file_change_reviews

- file: reference to which file change this review is for
- change review: reference to the overall change review that contains this
file review; every file review belongs to exactly one overall review
- details - full narrative detail if needed to expand on the description
- description: narrative paragraph if needed to extend the summary
- why: justification of the file change
- summary: 65 character or less summary of the file change with respect to
the current submitted change pointer, semantically
- body summary: a non-empty snapshot of the associated body summary at review
time, 65 characters or less

there is at most one file_change_review per file_change

The referenced change review and file change must belong to the same change
sequence, and the file change must be one of the files covered by that change
review.

all fields must be set, but details can be empty, and description can be empty
if details is empty, the rest must be non-empty. Constraints not easily defined
in the db should be handled by app.

## files

Files have stable identities independent of paths.

- current_filepath: reference to the current filepath; null when the file is
  deleted

The current_filepath reference is the path at the workspace pointer, is
unique among live files, and is null when deleted at the workspace pointer.
Resolve paths at submitted or implemented pointers from file_changes history.
Migrations must preserve existing file identities and map existing paths to
filepaths entries.

## filepaths

An append-only registry of unique file paths represented by a directory and
filename. A filepath row is never updated or deleted; a path can be reused by a
new file identity after deletion.

- dir: reference to the directory containing the file
- name: filename within that directory

The `dir` and `name` combination must be unique. Names must be a single
non-empty path segment with no whitespace or separators, use ASCII letters in
either case, digits, '.', '_', and '-', and not equal `.` or `..`.

## dirs

Directories have unique project-relative paths. The empty path represents the
project root. Specs are under `.4sight/`; tracked text assets are under their
owning command or subcommand source directory. Nested text asset paths are
represented by directory rows for each path. Binary assets are not represented
in the spec database.

- path: unique directory path

Paths must contain no whitespace, use ASCII letters in either case, digits,
'/', '.', '_', and '-', contain no repeated '/' separators, and have no empty,
'.', or '..' segments. The `.4sight` directory is a valid leading segment.
