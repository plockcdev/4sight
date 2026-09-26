# 4sight spec change review agent prompt

Overview: render an output prompt for agent to have the agent
review submitted spec change for quality.

Follow instructions below, organized into sections

## Verify In Sync

Scan all command dirs and schema dir for supported spec filenames and scan db for
all "workspace" bodies.
For each, check if the on disk text matches db body, or missing a counterpart.

Any out of sync files not indicated in the prompt as included in the change
requires the output prompt to tell the user the per file status and how to remediate
(e.g. overwrite local files with db body, or do a change import), then output prompt should tell agent to stop.

## Verify new change

Verify that the current workspace pointer is not the same as the
submitted pointer, if it is, then output prompt should tell agent to stop, giving instructions to user that they should submit
a request prompt to change specs or to import spec changes.

## Render meta info

Render the id of the current workspace change, the current submitted change, and the prompt (if not empty).

## Calculate Diff From Submitted

In a temporary file, create a unified diff covering each file in the current submitted
change and in the current workspace change by treating the submitted as the old and
the workspace as the new, effectively aggregating the diff of each change between the two
pointers.

## Calculate diffs

Render to output prompt all file spec paths relative to .4sight/ in a "Workspace Specs"
section

Then render the path to the temporary diff file created in earlier section (the diff
of the submitted change to the workspace change).

Output prompt instructs agent to read the entire db/file diff file to see all the changes since the last submitted spec, read the full spec of any file referenced in the diff,
and read any spec files related to the change prompt such as db spec files and command parent design specs.

Then output prompt instructs agent to make changes to specs on
disk (creating subcommand dirs/spec files as needed) based on the
user prompt in the preamble, if there was any (no prompt means
no additional changes).

If there were spec changes due to prompt, output prompt should
tell agent to run `4sight spec change import` to pick up the changes, and it will move the current workspace pointer.

Tell agent to submit a qualitative review of the spec changes
compared to the submitted change (including any due to the prompt), running
`4sight spec change review agent response`
The agent should stream
a json body providing all expected data.
