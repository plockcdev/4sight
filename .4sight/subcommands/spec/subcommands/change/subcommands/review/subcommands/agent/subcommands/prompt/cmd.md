# 4sight spec change review agent prompt

Overview: It is expected that an agent will call this command, read the output,
and follow the output instructions as a prompt. This commands intent is to have
the agent review the workspace change for quality.

If the current workspace change already has a review, an ordinary review
request must stop and tell the user that a review already exists, asking them
to explicitly request a re-review. For an explicit re-review request, continue
even when a review exists, include the user's new information about the prior
review or discovered mistake in the review context, and identify the next
attempt number. Do not alter prior attempts.

Review the specifications for clarity, completeness, and consistency. They
describe intended behavior, which may not yet be implemented; do not treat
missing or incomplete code implementation as a specification defect.
Check that schema specs contain persisted data definitions and constraints,
while application design and command specs specify operational choices such as
database location, initialization, and access. Flag misplaced requirements and
missing cross-spec behavior, including a location change that requires application
work even when it requires no SQL. For changed schemas, assess whether migration
requirements and any no-SQL conclusion are clear enough to implement safely;
do not infer equivalence merely from descriptive wording or checksums.
Check that newly introduced managed databases can use the standard application
migration tracking requirements in the schema spec.
Tell the reviewer Dolt is the default when no different database is established,
using file-backed access without a daemon, and other backends have not been
tested. Users should commit Dolt database files alongside application changes;
a Git-based rollback mechanism is deferred. Respect project-specific requirements. Treat
4sight's working database bootstrap and migration workflow as established.
For database-backed applications, assess the application initialization and
upgrade lifecycle against the authoring reference's default, respecting an
established alternative or an explicit exclusion of migration support. Do not
assume 4sight's working-database migrations upgrade deployed instances. Identify
missing application access, deployment lifecycle, or existing-data import
decisions when they matter to the specified behavior.

Embed the author command's `assets/application-database-lifecycle.md` at build
time and include it in the review prompt. Keep the embedded copy synchronized
with the source asset; do not require it in the target project.

The meta info is id of the current workspace change, the current submitted
change, and the change prompt (if not empty). Explicitly identify the workspace
change ID as the change being reviewed and require the agent to pass that exact
ID as the argument to the response command.

Command do all the following sections unless conditions require the agent and
command to stop.

## Verify new change

Verify that the current workspace pointer is not the same as the submitted
pointer, if it is show user the meta info and tell user to submit a request
prompt to change specs or to import spec changes, then stop.

## Verify In Sync

Scan supported spec files and tracked text assets using discovery and comparison
rules in the inherited spec design. Resolve file existence and deletion using
the schema spec. Compare disk and workspace state in
both directions.

Any out of sync files tell user the status for each out of sync file (including
+/- line count) and how to remediate (e.g. overwrite local files with db body,
or do a change import), then stop.

## Verify unreviewed changes

If the change prompt is non-empty, tell the user to run
`4sight spec change author agent prompt` to author the requested spec changes,
then stop. This command is review-only.

For an explicit re-review, do not stop merely because all changed files already
have reviews. Determine the next attempt number for this change and review the
full change again. For a first review, if the current workspace pointer has no
file changes missing reviews, tell the user there is nothing to review and stop.

## Render meta info

Render the meta info.

## Calculate Diff From Submitted

In a new file in `.4sight/changes/<workspace_change_id>/unified.diff`, create a
unified diff rooted in .4sight, covering each body in the current "submitted"
change pointer as old compared to each body for the new current workspace change
pointer, effectively squashing the diff of each change between the two pointers.
Include text asset changes, path changes, deletions, and files removed or added in the
diff. Resolve pointer-specific paths and path transitions using the file history
rules in the schema spec.

Call this the composite diff.

## Calculate diffs

Show agent all live workspace db spec paths in a "Current Workspace DB Specs"
section (these should all be in schemas subdir). Do not list assets or deleted
files.

Show agent all live workspace code spec paths in a "Current Workspace Specs"
section, except for the db spec files already printed. Do not list assets or
deleted files.

List code and schema specs that were deleted between the submitted and
workspace pointers. Include a spec only if its latest state at the submitted
pointer is live and its latest state at the workspace pointer is deleted.
Separate code specs and schema specs into different lists. Do not include
text assets in these lists; text asset deletions are covered in the composite diff and
file reviews.

Output prompt instructs agent to read the composite diff file to see all the
changes since the last submitted specification (user does not know what a
pointer is), to read the full workspace spec of any file referenced in the diff,
to read all the db spec files, and to read any workspace spec files related to
either changed files or the change prompt such as command parent design specs or
parent coding standards. Instruct the reviewer to follow applicable coding
standards, including referenced standards and command-path overrides, and assess
changed specifications against them, unless directly instructed otherwise by
the user.

Provide the path to the composite diff file.

## Response

Read the response schema if one exists and follow its constraints.

Tell agent to submit a qualitative review, which must assess the full range of
specification changes after the submitted pointer through the workspace pointer,
including earlier unsubmitted changes. Do not limit the overall review to the
latest change. The overall why, summary, description, details, and review must
reflect that full range. Include a file review for every file changed in that
range.

Instruct agent to include file-level reviews for every changed spec and text asset.
For selected renames and copies, include available similarity and copy source
provenance in the review context. Do not present inferred relationships as
confirmed identity. Also include an overall change review and a review for each
file updated since the "submitted" change pointer.

Require the agent to run `4sight spec change review agent response
<workspace_change_id>` with the complete review in the required response
command format with all its declared constraints. The argument must be the
workspace change ID rendered in the meta info. Check that the command succeeds.
After it succeeds, the agent must present the overall review and discuss
actionable overall and per-file findings as specified below.

## Discuss review feedback

After saving the review, show the change ID, attempt, readiness, and full
change-level review text. Provide `4sight spec change review show <change-id>
<attempt>` for the complete saved review. Then walk the user through actionable
findings and unresolved decisions from the overall and file-level reviews one
item at a time. For each item, present its exact saved wording and source,
relevant spec content when needed, and a specific suggested resolution. Allow
the user to discuss, approve, modify, reject, or defer the suggestion before
moving to the next item. Do not infer approval from silence. Keep track of each
decision and preserve the saved review unchanged.

Do not edit specs or import changes during this discussion. When the user
finishes or stops the walkthrough, gather the agreed fixes into one request,
including the feedback context, decisions, and constraints needed to author the
changes. Do not include rejected or deferred suggestions as approved fixes.
If there are agreed fixes, run `4sight spec change request <prompt>` with that
request, then run `4sight spec change author agent prompt` and follow its
instructions, including import and the subsequent review. If there are no
agreed fixes, stop without creating a request. If there are no actionable
findings or unresolved decisions, stop after presenting the review.
