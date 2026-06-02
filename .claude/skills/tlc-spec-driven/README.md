# TLC Spec-Driven — Refactored Edition

> **This is a community _refactor_ of the original "TLC Spec-Driven" skill.**
> Every bit of the original design, methodology, and structure is the work of **Felipe Rodrigues** and the **Tech Lead's Club** community. Full credit goes to them.
> This copy lives inside one project's `.claude/skills/` and has been adapted in-repo (see **What This Refactor Changed**). The upstream skill is untouched and remains the canonical source — install it with the command below.

**Plan and implement projects with precision. Granular tasks. Clear dependencies. Right tools. Zero ceremony.**

---

## Credits & Attribution

This skill is **not** an original creation. It is a derivative work — a refactor of:

| | |
| --- | --- |
| **Original skill** | TLC Spec-Driven (v2.0.0) |
| **Author** | Felipe Rodrigues — https://github.com/felipfr · https://linkedin.com/in/felipfr |
| **From** | Tech Lead's Club community — https://github.com/tech-leads-club |
| **License** | CC-BY-4.0 |

The refactor was done **in this repository only**, to adapt the examples to a Go codebase and to sharpen a few things (routing, tone, greenfield gate persistence). It does not change the original's methodology or claim authorship of it.

### Install the original (the canonical version)

```bash
npx @tech-leads-club/agent-skills install -s tlc-spec-driven
```

That command pulls the official, published skill from the Tech Lead's Club catalog — the real source of truth. If you want the unmodified skill, that is what you should install.

> **Heads-up:** running the install command above writes the canonical version **over** this folder. It will overwrite the refactor described here. If you reinstall, back up this copy first (or re-apply the changes).

---

## What This Refactor Changed

Everything below is a local adaptation on top of the original. The original behavior and structure are preserved; these are refinements:

1. **Stack-agnostic examples.** The original leaned on TypeScript/JavaScript-only snippets. Examples in `design.md`, `tasks.md`, and `code-analysis.md` were made language-neutral, and the `brownfield-mapping.md` test-signals were rewritten in Go idioms (`t.Parallel()`, `:memory:` SQLite, `go.uber.org/mock`) to match this repo's stack.
2. **A routing entry point.** A **"When This Skill Activates"** block was added to `SKILL.md` — a short orientation (load `.specs/` state → assess scope → route to a phase → confirm depth) so the agent has a clear first move instead of inferring one.
3. **Lighter ceremony.** The all-caps `MANDATORY` / `CRITICAL` / `NEVER` imperatives in `implement.md`, `tasks.md`, and `validate.md` were reframed into reasoned guidance that explains the _why_. The genuinely load-bearing rules — test integrity above all — stay firm.
4. **Greenfield gate persistence.** On a brand-new project the agreed test types and gate commands are now written to `.specs/codebase/TESTING.md`, so the verification gates survive across sessions instead of being agreed verbally and then lost.
5. **README clarity.** Added a "when _not_ to over-spec" decision guide and a "what gets created, and when" lifecycle section (both retained below).
6. **`/spec:*` slash commands.** Thin command shims under `.claude/commands/spec/` were added as explicit entry points that route into this skill while preserving its auto-sizing. See **Slash Commands**.

---

## What Is This Skill?

TLC Spec-Driven changes how an AI agent approaches software work. Instead of a rigid, bureaucratic pipeline, it uses **4 adaptive phases** that auto-size to complexity — full rigor for complex features, near-zero ceremony for simple ones:

```
SPECIFY  →  DESIGN  →  TASKS  →  EXECUTE
required    optional*  optional*  required

* auto-skipped when the scope doesn't need it
```

**The complexity lives in the system, not in your workflow.** You talk naturally; the skill decides how deep to go:

| Scope | What happens |
| --- | --- |
| **Small** (≤3 files) | Quick mode — describe → implement → verify → commit |
| **Medium** (clear feature) | Specify → Execute (design and tasks stay inline) |
| **Large** (multi-component) | Full pipeline with formal design and task breakdown |
| **Complex** (ambiguity, new domain) | Full pipeline + gray-area discussion + research + interactive UAT |

When the skill activates, it first loads any existing `.specs/` state, assesses scope, routes to the right phase, and — when scope is ambiguous — states its read and lets you correct before investing.

---

## When To Use It — And When Not To Over-Spec

This is **not** a "write a giant spec for everything" tool. Auto-sizing means you bring _every_ change here and the skill picks the ceremony — the only real mistake is forcing the full pipeline onto a one-line change. A bug fix or a small refactor should **never** become a formal spec.

| Your situation | Start with | What actually happens |
| --- | --- | --- |
| One-line bug fix, config change, copy tweak | `/spec:change` | **No spec.** Describe → implement → verify → commit. ≤3 files. |
| Small, well-understood refactor (rename, extract, move) | `/spec:change` | Same express lane — small refactors are quick tasks, **not** specs. |
| A clear, self-contained feature | `/spec:feature` | Brief spec, then implement. Design and tasks stay inline. |
| A multi-component feature, or a large/risky architectural refactor | `/spec:feature` | Full pipeline: spec → design → atomic tasks → execute. |
| Ambiguous feature in a new domain | `/spec:feature` | Full pipeline + gray-area discussion + research + interactive UAT. |

**Rule of thumb:** one sentence and ≤3 files → Quick mode, don't spec it. Multiple user stories or real design decisions → spec it. A big architectural refactor is the _only_ refactor that earns a full spec; everything smaller goes through Quick mode.

---

## Quick Start

The skill works through **natural conversation** — these are trigger phrases, not strict syntax. Talk to the agent like a colleague ("I want to build auth", "fix the login, it returns 401"). The `/spec:*` slash commands below are explicit shortcuts for the same intents.

| What you want | Say this | Or use |
| --- | --- | --- |
| Start a new project | "Initialize project" | `/spec:init` |
| Work with existing code | "Map codebase" | `/spec:map` |
| Plan a feature | "Specify feature [name]" | `/spec:feature` |
| Small change / bug fix | "Quick fix: [description]" | `/spec:change` |
| Resume previous work | "Resume work" | `/spec:resume` |
| Checkpoint before stopping | "Pause work" | `/spec:pause` |

---

## Slash Commands (`/spec:*`)

This refactor adds explicit command shortcuts in `.claude/commands/spec/`. Each is a **thin shim** — it routes into this skill's relevant phase and preserves auto-sizing (e.g. `/spec:change` will recommend `/spec:feature` if the work outgrows 3 files, and `/spec:feature` will drop back to `/spec:change` if it turns out trivial). They don't re-implement any logic, so they can't drift from the skill.

| Command | Does |
| --- | --- |
| `/spec:feature` | Plan & specify a feature (enters Specify, auto-sizes from there) |
| `/spec:change` | Small change — bug fix, small refactor, or config — via Quick mode (no spec) |
| `/spec:resume` | Load the last handoff + STATE and continue |
| `/spec:pause` | Write a session handoff before stopping |
| `/spec:map` | Brownfield analysis of an existing codebase → `.specs/codebase/` |
| `/spec:init` | Initialize the project (PROJECT.md: vision, goals, roadmap) |

> Commands are project-local. You may need to reload the agent for new commands to appear, and the exact menu label may depend on your tool's version.

---

## Project Structure

The skill creates a `.specs/` directory to organize all project documentation:

```
.specs/
├── project/
│   ├── PROJECT.md      # Vision, goals, tech stack, constraints
│   ├── ROADMAP.md      # Milestones, features, status tracking
│   └── STATE.md        # Persistent memory: decisions, blockers, lessons, todos, deferred ideas
│
├── codebase/           # Brownfield analysis (existing projects); greenfield persists TESTING.md here too
│   ├── STACK.md        # Technology stack and dependencies
│   ├── ARCHITECTURE.md # Patterns, data flow, code organization
│   ├── CONVENTIONS.md  # Naming, style, coding patterns
│   ├── STRUCTURE.md    # Directory layout and modules
│   ├── TESTING.md      # Test frameworks, coverage matrix, gate commands
│   ├── INTEGRATIONS.md # External services and APIs
│   └── CONCERNS.md     # Tech debt, risks, fragile areas
│
├── features/           # Feature specifications
│   └── [feature-name]/
│       ├── spec.md     # Requirements with traceable IDs (FEAT-01, AUTH-02...)
│       ├── context.md  # User decisions for gray areas (only when needed)
│       ├── design.md   # Architecture and components (only for large/complex)
│       └── tasks.md    # Atomic tasks with dependencies (only for large/complex)
│
└── quick/              # Ad-hoc tasks (quick mode)
    └── NNN-slug/
        ├── TASK.md     # Description + verification
        └── SUMMARY.md  # What was done + commit
```

### What Gets Created — And When

**First run:** the skill creates the `.specs/` directory at your repository root. What lands there depends on your first move:

- `/spec:init` ("Initialize project") → `.specs/project/PROJECT.md` (and `ROADMAP.md` once you ask for a roadmap)
- `/spec:map` ("Map codebase", existing code) → the analysis docs in `.specs/codebase/`

**Created once, then kept current (singletons — you don't regenerate these):**

- `project/PROJECT.md`, `project/ROADMAP.md` — vision and plan
- `project/STATE.md` — persistent memory; the skill _appends_ decisions, blockers, lessons, and deferred ideas as you work
- `codebase/*` — brownfield analysis; refreshed only when the code changes materially. On greenfield projects, `codebase/TESTING.md` is created the first time tasks are planned, to lock in the test types and gate commands.

**Created fresh for each unit of work (this is how you "make more than one"):**

- Every feature gets its **own** folder — `.specs/features/<feature-name>/`. A second feature is just a second folder; they never share a spec. Work one feature at a time.
- Every quick task gets its own numbered folder — `.specs/quick/NNN-slug/`.

In short: project- and codebase-level docs are created **once and updated**; each new feature or quick fix spins up its **own isolated folder**.

---

## The Four Adaptive Phases

### Specify (always)

**Goal:** capture WHAT to build with testable, traceable requirements.

The agent acts as a thinking partner, not an interviewer — it asks clarifying questions, challenges vagueness, and captures requirements with traceable IDs:

```markdown
### P1: User Login ⭐ MVP

**User Story:** As a user, I want to log in so that I can access my account.

| Requirement ID | Acceptance Criteria                                                            |
| -------------- | ------------------------------------------------------------------------------ |
| AUTH-01        | WHEN user enters valid credentials THEN system SHALL authenticate and redirect |
| AUTH-02        | WHEN user enters invalid credentials THEN system SHALL display error message   |
```

**Discuss gray areas (auto-triggered):** when the spec has ambiguous, user-facing decisions (layout, interaction patterns, error-handling style), the agent asks about them and records the answers in `context.md` — locking those decisions before design. It is not a separate phase; it only fires within Specify when ambiguity is detected.

### Design (when needed)

**Goal:** define HOW to build it — architecture, components, what to reuse.

**Skipped when** the change is straightforward (no architectural decisions, no new patterns). For simple features, design happens inline during Execute.

**Includes research:** before designing with unfamiliar tech, the agent follows the **Knowledge Verification Chain** (codebase → project docs → Context7 MCP → web search → flag as uncertain). It never assumes or fabricates — if it can't find documentation, it says so.

**Output:** `design.md` with architecture, component definitions (described in your stack's idiom), and integration points.

### Tasks (when needed)

**Goal:** break work into granular, atomic tasks with clear dependencies.

**Skipped when** there are ≤3 obvious steps — those get listed inline at the start of Execute.

**Safety valve:** if listing inline steps reveals >5 steps or complex dependencies, the agent stops and creates a formal `tasks.md`, acknowledging the Tasks phase was wrongly skipped.

| Vague task | Atomic tasks |
| --- | --- |
| "Create form" | T1: Create email input · T2: Add email validation · T3: Create submit button · T4: Add form state |

Each task includes: What (deliverable), Where (file path), Depends on, Reuses, Requirement (traceable ID), Done when (verifiable criteria + gate command), and the commit message.

### Execute (always)

**Goal:** implement one task at a time. Verify. Commit. Repeat.

```
Plan → Implement → Verify → Commit → Next
```

- **Surgical changes** — only touch required files
- **No scope creep** — if it's not in the task, don't touch it; capture ideas in STATE.md as Deferred Ideas
- **Verify before commit** — run the task's gate check; the test runner decides "done", not self-assessment
- **Atomic git commits** — one task = one commit ([Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/))

```
feat(auth): add email validation to login form
refactor(api): extract token refresh logic into service
fix(cart): prevent negative quantity on item decrement
```

**Feature-level validation** runs after all tasks complete — acceptance-criteria checks, code-quality review, and optionally interactive UAT for complex user-facing features.

---

## Quick Mode

For small tasks (bug fixes, config changes, tweaks ≤3 files) that don't need the pipeline:

```
You: /spec:change login returns 401 because token refresh skips the expired check

Agent: Quick Task: fix token refresh expired check
       Files: path/to/auth_service
       Approach: validate expiry before attempting refresh
       Verify: login with an expired token returns a new session, not 401

       [implements...]

       ✅ Committed: fix(auth): add expiry check to token refresh
```

**Guardrails:** max 3 files, max ~1 hour, no design decisions, no new dependencies. Exceed any of these and the agent recommends the full pipeline (`/spec:feature`).

---

## Complete Command Reference (trigger phrases)

These help the agent recognize intent — you don't need them verbatim, and each has a `/spec:*` shortcut.

### Project-level

| Trigger phrase | Description |
| --- | --- |
| "Initialize project", "Setup project" | Create PROJECT.md (vision, goals, constraints) |
| "Create roadmap", "Plan features" | Create ROADMAP.md (milestones and features) |
| "Map codebase", "Analyze existing code" | Create the brownfield docs for an existing project |
| "Document concerns", "Find tech debt" | Identify and document codebase risks |
| "Record decision", "Log blocker", "Add todo" | Add entries to STATE.md |
| "Pause work", "End session" | Create a handoff for session continuity |
| "Resume work", "Continue" | Load previous state and continue |

### Feature-level (auto-sized)

| Trigger phrase | Description |
| --- | --- |
| "Specify feature", "Define requirements" | Create spec.md with requirement IDs |
| "Discuss feature", "How should this work" | Capture user decisions for gray areas |
| "Design feature", "Architecture" | Create design.md |
| "Break into tasks", "Create tasks" | Create tasks.md with atomic breakdown |
| "Implement task", "Build", "Execute" | Execute a task with verification |
| "Validate", "Verify work", "UAT" | Feature-level validation and testing |
| "Quick fix", "Quick task", "Bug fix" | Express lane for small changes |

---

## Workflow Examples

### Implementing a feature (auto-sized)

```
You: /spec:feature user-authentication

Agent: [asks clarifying questions, creates spec.md with requirement IDs]
       I see gray areas in the UX — how should failed login attempts behave?
       [discusses, creates context.md]

You: Design the feature
Agent: [researches via the Knowledge Verification Chain, creates design.md]

You: Break into tasks
Agent: [creates tasks.md with atomic tasks + parallel execution plan]

You: Implement T1
Agent: Implementing T1: create auth service interface
       Files: path/to/auth_service
       Verify: compiles, contract exported
       [implements...]
       ✅ Committed: feat(auth): create auth service interface
       Next available: T2, T3 [P] (parallel)
```

### Session continuity

```
You: /spec:pause
Agent: ✅ Created .specs/HANDOFF.md — feature: user-authentication,
       completed T1/T2, T3 in progress (70%), next: finish T3 then T4

--- later, new session ---

You: /spec:resume
Agent: Resuming user-authentication at T3
       Completed: T1, T2 · In progress: T3 (auth service)
       Next: finish T3, then T4 (parallel with T5). Continue with T3?
```

---

## Context Management

| Load strategy | Documents | Tokens |
| --- | --- | --- |
| **Base load** (always) | PROJECT.md, ROADMAP.md, STATE.md | ~15k |
| **On-demand** | Current spec, context, design, or tasks | +5–10k |
| **Never simultaneous** | Multiple feature specs or architecture docs | — |

**Target:** <40k tokens loaded. **Reserve:** plenty of headroom for work, reasoning, and outputs. When context grows large, the skill shows a status indicator and suggests optimizations.

---

## Skill Integrations

TLC Spec-Driven works better alongside complementary skills. It detects whether they're installed and delegates; if not, it falls back gracefully and recommends them once per session.

| Skill | Integration |
| --- | --- |
| **mermaid-studio** | Diagrams — architecture overviews, data flows, sequence diagrams |
| **codenavi** | Code exploration — brownfield mapping, pattern identification, dependency tracing |

---

## Reference Files

Loaded on demand:

| File | Purpose |
| --- | --- |
| `project-init.md` | Project initialization process and template |
| `roadmap.md` | Roadmap creation and milestone tracking |
| `brownfield-mapping.md` | Codebase analysis (the codebase docs) |
| `concerns.md` | Tech debt, risks, and fragile areas |
| `specify.md` | Requirements gathering with traceable IDs |
| `discuss.md` | Gray-area discussion and context capture |
| `design.md` | Architecture, research, and component design |
| `tasks.md` | Granular task breakdown methodology |
| `implement.md` | Execute: implementation + verification + atomic commits |
| `validate.md` | Feature validation and interactive UAT |
| `quick-mode.md` | Express lane for ad-hoc tasks |
| `session-handoff.md` | Pause/resume work process |
| `state-management.md` | Persistent memory |
| `coding-principles.md` | Behavioral guidelines for implementation |
| `context-limits.md` | Token budget and monitoring |
| `code-analysis.md` | Search/analysis tools and fallbacks (language-neutral) |

---

## Tips

**Do**

- Start with project initialization, even for existing codebases
- Be specific about scope — clear boundaries prevent creep
- Trust the auto-sizing — let the agent apply the right depth
- Say "pause work" before ending — enables seamless resumption
- Challenge the agent — if something looks wrong, say so

**Don't**

- Force all phases — let the agent skip what's unnecessary
- Work on multiple features at once — one feature per cycle
- Skip verification — even quick tasks need a verify step
- Accept vague answers — ask for specifics

---

## FAQ

**Is this the official skill?**
No. This is a refactored copy. For the official, unmodified skill, install it with `npx @tech-leads-club/agent-skills install -s tlc-spec-driven`. All original credit belongs to Felipe Rodrigues and the Tech Lead's Club community.

**Can I skip phases?**
Yes — the skill auto-sizes. Design and Tasks are skipped for simple features; Quick mode skips the whole pipeline for small changes.

**Can I use this for small tasks or quick fixes?**
Yes — use `/spec:change` (or "Quick fix: …"). You get quality guardrails (verify + atomic commit) without the planning overhead.

**Does it work with any tech stack?**
Yes — it's stack-agnostic. (This refactored copy uses Go-flavored examples in places because it lives in a Go repository, but the methodology is language-agnostic.)

**What if the agent invents an API that doesn't exist?**
It follows a strict Knowledge Verification Chain (codebase → project docs → Context7 MCP → web search → flag as uncertain) and never fabricates. If it can't find documentation, it says "I don't know."

---

## License

CC-BY-4.0 © [Tech Lead's Club](https://github.com/tech-leads-club). Original skill by [Felipe Rodrigues](https://github.com/felipfr). This refactor preserves that license and attribution.
