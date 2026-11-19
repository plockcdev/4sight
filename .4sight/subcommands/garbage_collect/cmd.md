# 4sight garbage_collect [trash_name]

Permanently delete completed trashed application databases, retaining their audit
records. With no argument select all uncollected completed retirements; with an
argument select the exact trash database name. Reject an unknown, already
collected, or incomplete selection with a diagnostic. Never select an active or
internal database. If there are no eligible databases, report that and exit
successfully without prompting.

Before deletion, show each selected original and trash name, trash date, deleting
migration ID, and basic available size statistics (database size and table count);
label unavailable statistics as unavailable. Show the number selected and warn
that deletion is permanent. Require an interactive explicit `yes` confirmation
after the list; any other response, EOF, or unavailable interactive input cancels
without deletion. There is no implicit confirmation or noninteractive bypass.

Serialize with retirement and other collections, and revalidate each selected
record and physical database identity before deleting. Do not delete an unrelated
or untracked database. Record the successful collection time for each database;
retain earlier successful results if a later deletion fails, report per-database
results and return failure on any error. Use durable intent and reconciliation so
an interruption between physical deletion and recording success can be retried
safely. If the physical database is unexpectedly absent without a recorded
collection operation, stop and report it for repair rather than claim success.
