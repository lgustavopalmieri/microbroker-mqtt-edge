---
description: Small change — bug fix, small refactor, or config — via Quick mode (no spec)
argument-hint: [what to change, one sentence]
---

Use the **tlc-spec-driven** skill in **Quick mode** for a small change: **$ARGUMENTS**

This is the express lane for bug fixes, small refactors, and config tweaks — no spec, no pipeline. Follow quick-mode:

1. Restate it as a one-line task, then list the files to touch (≤3), the approach, and how you'll verify it.
2. Wait for my OK. **Then apply the Skill Gate before writing any code** (same rule as `/ship`, scaled to the small scope) — your own judgment does not override it:
   - touches Go tests → use **`test-expert`** (two-phase: list cases for my approval first, then implement).
   - touches goroutines / channels / sync / tickers / shared state → consult **`go-concurrency-patterns`** and keep it `-race` clean; never ship synchronous, unguarded code for a concurrent change.
   - adds or alters SQL schema / a migration → use **`add-migration`**.
   - new module/feature skeleton → use **`hexagonal-scaffold`**.

   If none apply (pure config/doc tweak), say so and proceed. Then implement surgically (touch only the listed files), verify, and make one atomic commit.
3. Record it under `.specs/quick/NNN-slug/` and add a row to STATE.md.

If the pre-check shows it needs >3 files, design decisions, or new dependencies, stop and recommend `/spec:feature` instead — don't quietly grow the scope.
