# 4sight upgrade

Upgrade an existing initialized project to the running binary's internal database
version and refresh repeatable managed setup. This command takes no arguments.
It is safe to repeat, including when already current. Never initialize a new or
partially initialized project, recreate a missing internal database or tracking
table, or apply application database migrations. On an uninitialized project,
instruct the user to run `4sight init`; on partial or damaged state, stop and
report recovery guidance without mutation.

Validate both embedded migration chains, expected IDs, and both recorded
histories according to the shared preflight. A valid older complete prefix is
eligible for upgrade; an unknown, divergent, incomplete, newer, or otherwise
invalid history is an error and must not be changed. Serialize with other
upgrades, app migrations, and commands accessing internal databases; revalidate
under the lock before mutation. Apply missing embedded internal migrations in
order, first `4sight_migration`, then `spec`, recording each successfully applied
ID. Use atomic migration and tracking updates where supported. Otherwise report
that a failure may require repair, stop, and do not claim success; retry only when
the recorded and actual state can safely be reconciled. Preserve application
migration records and do not alter application databases, app migration files,
or their current/submitted pointers. Internal history is authoritative in `4sight_migration.4sight_migration` and
`4sight_migration.spec`; compare it with embedded chains, not symlinks. Do not
write internal migration SQL files or create internal symlinks.

In an ordinary application project, ignore and leave untouched any legacy
internal migration files and symlinks, including `current`; do not require or
reconcile them. Determine project identity using the rules in the spec design;
if identity requires a user answer, stop and ask before mutation.

When 4sight is developing itself, existing internal `current` symlinks represent
source migration state, not authoritative applied history. If one is present,
reconcile it after each recorded migration to the corresponding existing local
migration file, but only after verifying its filename, parent chain, and bytes
match the embedded migration. Never follow or replace an unsafe link. If the
file is missing or differs, leave the link alone, report the conflict, and fail
without claiming success; do not apply more migrations. When retrying after an
interruption, derive the last applied ID from the tracking table and repair a
stale valid link before applying more migrations. A recorded migration must not
be replayed merely to repair a link. If no internal current symlink exists,
none is required or created.

After successful migration, apply the repeatable setup rules of `init`: ensure
`.4sight/changes` is in `.gitignore`, maintain the managed root AGENTS section,
overwrite `.4sight/AGENTS.md` and `.4sight/coding_suggestions.md` with their
current managed assets, and create `.4sight/coding.md` only if absent. Preserve unrelated content. Do not reimport
specs or reset workspace, submitted, implemented, or review history. Do not stage
or commit files. If maintenance fails, report what succeeded and what needs repair;
a retry must be safe. Report whether migrations and managed files changed.

The binary embeds internal migration chains through its expected migration IDs;
use the build-time migration assets described by `init`. Help and usage explain
when to use `init`, `upgrade`, and `spec change implement agent migrate up`.
