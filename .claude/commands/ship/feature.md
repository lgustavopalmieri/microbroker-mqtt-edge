---
description: Ship the next pending task of a spec feature — implement, gate, atomic commit, mark done
argument-hint: [feature name] (optional; defaults to the active feature in STATE.md)
---

You are a **standalone task runner** for spec-driven features. Do **NOT** load the `tlc-spec-driven`
skill — operate directly on the spec artifacts under `.specs/`.

**Target feature:** `$ARGUMENTS` — if empty, read the active feature from `.specs/project/STATE.md`.

## Protocol — ship exactly ONE task, then STOP

1. **Locate.** Read `.specs/features/<feature>/tasks.md` (especially its **Progress Ledger**),
   `.specs/features/<feature>/design.md`, and `.specs/codebase/TESTING.md`. If there is no `tasks.md`,
   tell me the feature hasn't been broken into tasks yet (`/spec:feature`) and stop.

2. **Select the next task.** Pick the lowest-ID **⬜ Pending** task whose **every dependency is ✅ Done**.
   If nothing is actionable (all done, or remaining tasks are blocked), report the ledger state and stop.
   Announce which task you picked and why.

3. **Preflight.** Working tree must be clean except `.specs/**`. If on the default branch (`develop`),
   create/switch to a feature branch `feat/<feature>` before touching code. Set the task to **🔄 In progress**
   in the ledger.

4. **Implement** only that task's deliverable — its What / Where / Reuses — obeying the repo rules
   (CLAUDE.md hexagonal layout, dependencies point inward, pure-Go SQLite `CGO_ENABLED=0`, conventions).
   **Co-locate its tests** per the task's `Tests` field (never defer to a later task). The skills
   `hexagonal-scaffold`, `add-migration`, `test-expert`, `go-concurrency-patterns` are available as helpers.

5. **Gate.** Run the task's gate from TESTING.md (`quick` / `full` / `build`). On failure, fix within this
   task's scope. If you cannot get it green, revert/stash the code changes, set the task back to ⬜ Pending,
   report the failure, and stop — **never commit red**.

6. **Atomic commit.** One well-scoped commit that tells *this task's* story, using the task's predefined
   Conventional-Commit message. If the task is genuinely large you MAY split into a few logical commits —
   but each must build and pass, and the set must be self-contained to this task. End every commit message with:
   `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`

7. **Mark done.** Flip the task to **✅ Done** in the Progress Ledger, update the matching `spec.md`
   traceability `Status`, and update `.specs/project/STATE.md` (active work + next actionable task). Include
   these marker edits in the task commit (or a trailing `chore(spec): mark <Tn> done`).

8. **Report & STOP.** Summarize: task shipped, files changed, gate result, commit hash(es), and the next
   actionable task. Do **not** start the next task.

If `/loop` is driving you, ship one task per tick with this same protocol.
