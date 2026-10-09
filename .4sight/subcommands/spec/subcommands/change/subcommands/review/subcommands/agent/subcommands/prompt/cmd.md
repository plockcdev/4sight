# 4sight spec change review agent prompt

Overview: It is expected that an agent will call this command, read the output,
and follow the output instructions as a prompt. This commands intent is to have
the agent review the workspace change for quality.

The meta info is id of the current workspace change, the current submitted
change, and the change prompt (if not empty).

Command do all the following sections unless conditions require the agent and
command to stop.

## Verify new change

Verify that the current workspace pointer is not the same as the submitted
pointer, if it is show user the meta info and tell user to submit a request
prompt to change specs or to import spec changes, then stop.

## Verify In Sync

Scan all command dirs and schema dir for supported spec filenames and scan db
for all "workspace" bodies. For each, check existance and contents for both.

Any out of sync files tell user the status for each out of sync file (including
+/- line count) and how to remediate (e.g. overwrite local files with db body,
or do a change import), then stop.

## Verify Prompt or unreviewed changes

If the prompt is not empty, continue to next section.

If the current workspace change pointer has no file changes missing reviews,
then tell the user there is nothing to review and stop.

## Render meta info

Render the meta info.

## Calculate Diff From Submitted

In a new file in `.4sight/changes/<workspace_change_id>/unified.diff`, create a
unified diff rooted in .4sight, covering each body in the current "submitted"
change pointer as old compared to each body for the new current workspace change
pointer, effectively squashing the diff of each change between the two pointers.
Include files removed and added in the diff.

Call this the composite diff.

## Calculate diffs

Show agent all workspace file db spec paths relative to .4sight/ in a "Current
Workspace DB Specs" section (these should all be in schemas subdir). Do not list
empty files.

Show agent all workspace file spec paths relative to .4sight/ in a "Current
Workspace Specs" section, except for the db spec files already printed. Do not
list empty files.

List specs whose latest body at the submitted pointer is non-empty and whose
latest body at the workspace pointer is empty or absent. These are the specs
removed in this change. Separate DB specs and code specs as different lists.

Output prompt instructs agent to read the composite diff file to see all the
changes since the last submitted specification (user does not know what a
pointer is), to read the full workspace spec of any file referenced in the diff,
to read all the db spec files, and to read any workspace spec files related to
either changed files or the change prompt such as command parent design specs or
parent coding standards.

Provide the path to the composite diff file.

## User Prompt Handling

We only want to render a user prompt for the workspace change if it is
non-empty, and do so at the end of the file after a "---" line acting as a
document separator.

For non-empty user prompt, instruct agent to make changes to specs on disk
(creating subcommand dirs/spec files as needed) based on the user prompt.

Note: the user prompt is for making spec changes of the current workspace change
pointer, user prompt does not get reviewed, the specs get reviewed.

## Importing Spec Changes Due to User Prompt

If there were spec changes due to user prompt, tell agent to run `4sight spec
change import` to pick up the changes, and it will move the current workspace
pointer.

If there were no spec changes due to user prompt, tell user no review is needed
for the prompt as there were no spec changes, then stop.

## Response

Read the response schema if one exists and follow its constraints.

Tell agent to submit a qualitative review, which must assess the full range of
specification changes after the submitted pointer through the workspace pointer,
including earlier unsubmitted changes. Do not limit the overall review to the
latest change. The overall why, summary, description, details, and review must
reflect that full range. Include a file review for every file changed in that
range.

Instruct agent to also include overall change review and a review for each file
updated since the "submitted" change pointer.

Require the agent to run `4sight spec change review agent response` with the
complete review in the required response command format with all its declared
constraints. Check that the command succeeds. After it succeeds, the agent must
include the overall review and per-file findings in its user-facing reply.
