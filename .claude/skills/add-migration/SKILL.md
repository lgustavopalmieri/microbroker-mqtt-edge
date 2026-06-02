---
name: add-migration
description: Scaffold the next database migration for this repo. Use when adding or altering a SQLite table/index/schema.
---

# add-migration

First decide which migration system is in play — the team plans to move from the custom embedded migrator to **goose/atlas**, so check before assuming:

```bash
ls internal/platform/database/migrations/        # custom migrator (current)
grep -rl "goose\|atlas" go.mod tools.go 2>/dev/null   # has the new tool landed?
```

## Current state — custom embedded migrator

If goose/atlas is **not** set up yet:

1. Look at the highest-numbered file in `internal/platform/database/migrations/` and pick the next zero-padded prefix (`001_` → `002_` → …).
2. Create `internal/platform/database/migrations/<NNN>_<short_snake_description>.sql` with the DDL. Statements are split on `;`, each migration runs in a transaction, and applied files are tracked in `schema_migrations` (see `internal/platform/database/migrator.go`).
3. **Migrations are immutable once committed** — never edit an existing numbered file; always add a new one. Migrations run automatically at broker startup.
4. Verify: `go build ./...` then `go test ./...` (the migrator and repository have tests under `internal/platform/database/`).

## After the move to goose/atlas

If goose/atlas **is** set up, use its CLI instead (`goose -dir <dir> create <name> sql`, or the atlas equivalent) and follow that tool's conventions — do not hand-create numbered files. Update this skill when the transition lands.
