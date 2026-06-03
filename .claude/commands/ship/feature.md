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

4. **Skill Preflight Gate — MANDATORY, before writing any code.** Read the task's `Tools` /
   `Skill` line. For **each** skill listed there (anything other than `NONE`), you **MUST** invoke it
   via the Skill tool *before* touching code — this is not optional and your own judgment does not
   override it. The mapping is mechanical, not discretionary:
   - `hexagonal-scaffold` → invoke it to generate the `domain → application(interface.go) → adapters`
     skeleton **before** filling in logic, so the layout matches the repo exactly.
   - `add-migration` → invoke it to scaffold the next numbered migration **before** writing any SQL.
   - `go-concurrency-patterns` → invoke it **before** writing any code that involves goroutines,
     channels, tickers, shared state, or a `-race` gate; use it to pick the concurrency pattern.
     Never ship synchronous, unguarded code for a task whose deliverable is inherently concurrent.
   - `test-expert` → invoke it for **all** tests, and follow its two-phase workflow in full: list the
     test cases for **my approval first**, then implement. Never hand-write tests that skip this.

   If you believe a listed skill genuinely does not apply, **stop and ask me** — do not silently skip it.
   Independently, if a task touches concurrency/migrations/tests but its `Skill` line omits the matching
   skill, surface that gap and invoke the skill anyway.

5. **Implement** only that task's deliverable — its What / Where / Reuses — obeying the repo rules
   (CLAUDE.md hexagonal layout, dependencies point inward, pure-Go SQLite `CGO_ENABLED=0`, conventions),
   building on the skill output from step 4. **Co-locate its tests** per the task's `Tests` field
   (never defer to a later task).

6. **Gate.** Run the task's gate from TESTING.md (`quick` / `full` / `build`). On failure, fix within this
   task's scope. If you cannot get it green, revert/stash the code changes, set the task back to ⬜ Pending,
   report the failure, and stop — **never commit red**.

7. **Atomic commit.** One well-scoped commit that tells *this task's* story, using the task's predefined
   Conventional-Commit message. If the task is genuinely large you MAY split into a few logical commits —
   but each must build and pass, and the set must be self-contained to this task. End every commit message with:
   `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`

8. **Mark done.** Flip the task to **✅ Done** in the Progress Ledger, update the matching `spec.md`
   traceability `Status`, and update `.specs/project/STATE.md` (active work + next actionable task). Include
   these marker edits in the task commit (or a trailing `chore(spec): mark <Tn> done`).

9. **Report & STOP.** Summarize: task shipped, files changed, **which skills you invoked in step 4**,
   gate result, commit hash(es), and the next actionable task. Do **not** start the next task.

If `/loop` is driving you, ship one task per tick with this same protocol.
