## 4sight spec chage review design

After spec changes have been imported, the agent will make a review agent
prompt. Request-driven spec authoring is handled by the separate `spec change
author agent prompt` command.

The review prompt instructs the agent to perform a high-level change review and
per-file qualitative reviews, then provide an agent response.

Each review of a change is a numbered attempt, starting at 1. A re-review
creates the next attempt and preserves earlier attempts, so the reviewer can
account for new information without erasing the prior opinion. An ordinary
review request must stop if the current workspace change already has a review;
only an explicit user request for a re-review proceeds to the prompt and
response workflow.

The review is for the current "workspace" change pointer with respect to the
current "submitted" change pointer while the filesystem and db agree on
contents.

The composite diff includes tracked text asset changes. Text asset files are omitted from
the lists of workspace specs, but every changed text asset receives a file-level
review entry. Use the inherited spec design and the schema spec for text asset,
path, and relationship semantics. Review must
distinguish inferred relationships from confirmed file identity.
