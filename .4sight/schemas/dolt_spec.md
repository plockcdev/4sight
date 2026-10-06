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

The change review for the root change should be created by migration with empty
strings for description/details:

- id: `ch-000`
why: to provide an anchor for all changes summary: initialization review:
perfect ready: true

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
- change: a reference for which change this file change is part of
- file: which file this diff is calculated for
- diff - what changed in unified diff format
- created_at: second resolution date and time

There can be only one file change per file per change, but a change can have
multiple file changes by pointing to different files

Conceptually, a file change is created based on the current workspace's parent
change (as the new workspace change must be created before any file changes),
which should match the current body text for the file as the file change has to
come before the body update.

If no previous file change found, baseline is empty string.

## file_change_reviews

- file change: reference to which file change this review is for
- details - full narrative detail if needed to expand on the description
- description: narrative paragraph if needed to extend the summary
- why: justification of the file change
- summary: 65 character or less summary of the file change with respect to
the current submitted change pointer, semantically

there is at most one file_change_review per file_change

all fields must be set, but details can be empty, and description can be empty
if details is empty, the rest must be non-empty. Constraints not easily defined
in the db should be handled by app.

## files

Files on disk, which for example could be part of a code or schema spec

- dir: reference to which dir this file is in
- name: no whitespace file name with combination of lowercase letters,
hyphens, underscores, numbers, and '.', but not starting with a hyphen or period

dir and name combination must be unique

## dirs

Filesystem dir path relative to .4sight, could for example be part of a change
or a spec

- id
- path: unique directory path, no whitespace with combination of lowercase letters,
hyphens, numbers, '/', and '.' if non-empty, must start with letter or number,
can't end with '/' or have two '/' in a row empty dir is allowed (root of
.4sight)
