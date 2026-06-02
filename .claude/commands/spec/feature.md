---
description: Plan & spec a feature with the tlc-spec-driven skill (auto-sized)
argument-hint: [feature name or short description]
---

Use the **tlc-spec-driven** skill to plan and specify a feature: **$ARGUMENTS**

Enter at the **Specify** phase and let the skill auto-size from there:

- Create or load `.specs/features/<feature>/spec.md` with traceable requirement IDs (e.g. `AUTH-01`).
- Trigger the gray-area discussion only if the spec has ambiguous, user-facing decisions.
- Continue into Design → Tasks → Execute **only to the depth the scope demands** — don't force phases the change doesn't need.

If this turns out to be a one-sentence, ≤3-file change, stop and tell me it's really a `/spec:change` (Quick mode), not a feature.
