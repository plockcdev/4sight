Do not treat CHANGELOG as a spec input

# 4sight spec driven implementation management of program

## Specification layout

Always read `.4sight/design.md` before working in this repository. It defines
the program design, command directory structure, and specification conventions.

# Implementing Schema Specs

If user asks you to implement a schema spec or "the schema", run `4sight migrate
implement prompt <path to schema spec relative to schemas dir>` and then follow
the prompt

Do not do this if they just ask to implement "the spec" Do not update any schema
specs or code specs as part of implementation

# Implementing Code Specs

If user asks you to implement code spec (or doesn't specify which kind of spec),
run `4sight spec implement prompt` and then follow the prompt

Do not update or import any code or schema specs during implementation

Before implementing code that depends on a schema change, check that
database’s current and submitted migration symlinks. If they point to
different migrations, stop and ask the user to run 4sight migrate up. Do not
update code to work with the pending schema until current matches submitted.

# Handling User request for spec review

If any un-imported spec changes exist in the workspace, stop after telling user
to import or reset the workspace specs.

When the user requests a spec review, run `4sight spec change review agent
prompt` and follow the instructions

If the current workspace change already has a review and the user asks for a
review, stop and tell them a review already exists. Ask them to request a
re-review explicitly. When they request a re-review, run `4sight spec change
review agent prompt` and follow the instructions for a new numbered attempt.

After review completes, show the change level review text and then the command
for user to view the review in detail

# Handling spec change requests

For later reference, check if there any un-imported spec changes in the
workspace.

When the user requests a change to the spec and provides instructions on those
changes, run `4sight spec change request <prompt>` where prompt is their request
and follow the instructions.

After following those instructions, if there were un-imported workspace changes
from earlier, then tell the user to either import the dirty spec or reset, then
return control to the user. Otherwise (prior clean spec workspace), follow the
instructions in the heading "Handling User request for spec review" to review
the change
