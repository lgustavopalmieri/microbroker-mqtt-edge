---
description: Ship a specific task of a spec feature by ID (for parallel team pickup)
argument-hint: <feature> <TaskID>   e.g. "availability T7"
---

You are a **standalone task runner** for spec-driven features. Do **NOT** load the `tlc-spec-driven`
skill — operate directly on the spec artifacts under `.specs/`.

**Target:** feature `$1`, task `$2` (e.g. `availability T7`). If either is missing, ask for it and stop.

## Protocol — ship exactly the named task, then STOP

1. **Locate.** Read `.specs/features/$1/tasks.md` (the **Progress Ledger** + the `$2` definition),
   `.specs/features/$1/design.md`, and `.specs/codebase/TESTING.md`. If `$2` doesn't exist, list the
   valid task IDs and stop.

2. **Guard.** Verify task `$2` is **⬜ Pending** — if it's ✅ Done, ask whether to redo it; if 🔄 In progress,
   warn it may be owned by someone else and ask before continuing. Check `$2`'s **dependencies**: if any is
   **not ✅ Done**, clearly warn which deps are unmet (its inputs/ports may not exist yet) and **ask for my
   confirmation** before proceeding — picking a task out of order is allowed but you must flag the risk.

3. **Preflight.** Working tree clean except `.specs/**`. If on the default branch (`develop`), create/switch
   to a branch `feat/$1` (or `feat/$1-$2` if you prefer per-task branches for parallel work) before touching
   code. Set `$2` to **🔄 In progress** in the ledger.

4. **Implement** only task `$2`'s deliverable — its What / Where / Reuses — obeying the repo rules (CLAUDE.md
   hexagonal layout, dependencies point inward, pure-Go SQLite `CGO_ENABLED=0`, conventions). **Co-locate its
   tests** per the task's `Tests` field. Helper skills: `hexagonal-scaffold`, `add-migration`, `test-expert`,
   `go-concurrency-patterns`.

5. **Gate.** Run `$2`'s gate from TESTING.md (`quick` / `full` / `build`). On failure, fix within scope; if you
   can't get green, revert/stash, set `$2` back to ⬜ Pending, report, and stop — **never commit red**.

6. **Atomic commit.** One well-scoped commit telling `$2`'s story, using its predefined Conventional-Commit
   message (a few logical commits allowed only if the task is large; each must build+pass). End every commit
   message with:
   `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`

7. **Mark done.** Flip `$2` to **✅ Done** in the Progress Ledger, update the `spec.md` traceability `Status`,
   and update `.specs/project/STATE.md`. Include these edits in the commit (or a trailing
   `chore(spec): mark $2 done`).

8. **Report & STOP.** Summarize: task shipped, files changed, gate result, commit hash(es), unmet-dependency
   warnings (if any), and the next actionable task. Do **not** start another task.
