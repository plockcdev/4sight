# 4sight spec change review agent prompt

Overview: It is expected that an agent will call this command, read the output,
and follow the output instructions as a prompt.  This commands intent is to have
the agent review the workspace change for quality.

The meta info is id of the current workspace change, the current submitted change, and the change prompt (if not empty).

Command do all the following sections unless conditions require the agent and command
to stop.

## Verify new change

Verify that the current workspace pointer is not the same as the
submitted pointer, if it is show user the meta info and tell user to submit
a request prompt to change specs or to import spec changes, then stop.

## Verify In Sync

Scan all command dirs and schema dir for supported spec filenames and scan db for
all "workspace" bodies.
For each, check existance and contents for both.

Any out of sync files tell user the status for each out of sync file
(including +/- line count) and how to remediate
(e.g. overwrite local files with db body, or do a change import), then stop.

## Verify Prompt or unreviewed changes

If the prompt is not empty, continue to next section.

If the current workspace change pointer has no file changes missing reviews, then tell the user there is nothing to review and stop.

## Render meta info

Render the meta info.

## Calculate Diff From Submitted

In a new file in `.4sight/changes/<workspace_change_id>/unified.diff`, create a unified diff rooted in .4sight, covering each body in the current "submitted"
change pointer as old compared to each body for the new current workspace change pointer, effectively squashing the diff of each change between the two
pointers.  Include files removed and added in the diff.

Call this the composite diff.

## Calculate diffs

Render to output prompt all workspace file spec paths relative to .4sight/ in a "Workspace Specs"
section

Then render the path to the composite diff file.

Output prompt instructs agent to read the composite diff file to see all the changes since the last submitted spec, read the full workspace spec of any file referenced in the diff,
and read any workspace spec files related to the change prompt such as db spec files and command parent design specs.

## User Prompt Handling

We only want to render a user prompt for the workspace change if it is non-empty,
and do so at the end of the file after a "---" line acting as a document separator.

For non-empty user prompt, instruct agent to make changes to specs on
disk (creating subcommand dirs/spec files as needed) based on the
user prompt.

Note: the user prompt is for making spec changes of the current workspace change pointer, user prompt does not get
reviewed, the specs get reviewed.

## Importing Spec Changes Due to User Prompt

If there were spec changes due to user prompt,
tell agent to run `4sight spec change import` to pick up the changes, and it will move the current workspace pointer.

If there were no spec changes due to user prompt, tell user no review is needed for the prompt
as there were no spec changes, then stop.

## Response

Tell agent to submit a qualitative review of all the spec changes of the current workspace change pointer (including any changes due to user prompt)
compared to the current submitted change pointer (including any due to the prompt), running
`4sight spec change review agent response`

Instruct agent to provide overall change review and a review for each
file updated since the "submitted" change pointer.
