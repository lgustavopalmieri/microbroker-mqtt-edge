# Code Analysis Tools

Use graceful degradation for code search and structural analysis.

## Tool Priority

1. **ast-grep** (`sg`) - Structural pattern-based search
2. **ripgrep** (`rg`) - Fast context-aware text search
3. **grep** - Standard text search (always available)

## Detection

Check tool availability before use:

```bash
# Check for ast-grep
if command -v sg >/dev/null 2>&1; then
  # Use ast-grep for structural search
elif command -v rg >/dev/null 2>&1; then
  # Fall back to ripgrep
else
  # Use standard grep as final fallback
fi
```

## Usage Examples

The examples below are written by **intent** (find definitions, imports, types), not for one language. Substitute your project's actual syntax: ast-grep is language-aware via `--lang`, while ripgrep/grep match raw text, so swap in the right keyword and file glob for the language you're searching.

**Definition keywords by language** (for the text-search fallbacks):

| Language | Function/method | Type/class                      | Import             |
| -------- | --------------- | ------------------------------- | ------------------ |
| Go       | `func`          | `type ... struct` / `interface` | `import`           |
| Python   | `def`           | `class`                         | `import` / `from`  |
| JS/TS    | `function`      | `class` / `interface` / `type`  | `import`           |
| Rust     | `fn`            | `struct` / `enum` / `trait`     | `use`              |
| Java/C#  | (return + name) | `class` / `interface`           | `import` / `using` |
| Ruby     | `def`           | `class` / `module`              | `require`          |

**Finding function/method definitions:**

```bash
# ast-grep (best — structural; set the language)
sg --lang <lang> -p '<function-definition-pattern>'   # e.g. --lang go -p 'func $NAME($$$) $$$'

# ripgrep (fallback — match your language's definition keyword)
rg '<keyword>\s+\w+\s*\(' -g '*.<ext>'                 # e.g. rg 'func \w+\(' -g '*.go'

# grep (last resort)
grep -rn '<keyword> ' --include='*.<ext>'
```

**Finding imports/dependencies:**

```bash
# ast-grep
sg --lang <lang> -p '<import-pattern>'                 # e.g. --lang ts -p 'import { $$$ } from "$MOD"'

# ripgrep
rg '^\s*<import-keyword>\b' -g '*.<ext>'               # e.g. rg '^\s*import\b' -g '*.go'
```

**Finding type / class / component definitions:**

```bash
# ast-grep
sg --lang <lang> -p '<type-definition-pattern>'        # e.g. --lang go -p 'type $NAME struct { $$$ }'

# ripgrep
rg '<type-keyword>\s+\w+' -g '*.<ext>'                 # e.g. rg 'type \w+ struct' -g '*.go'
```

## Search Scope

**Best practices:**

- Limit to source file extensions relevant to project
- Exclude directories: `node_modules`, `vendor`, `dist`, `build`, `.git`
- Focus on source directories: `src`, `lib`, `app`
- Use file type filters when available

**Performance tips:**

- Use specific patterns over broad searches
- Limit directory depth with `--max-depth` (ripgrep/grep)
- Cache results for repeated queries

## Fallback Notice

If ast-grep unavailable, display once per session:

```
⚠️ ast-grep not detected. Install for more precise structural code analysis.
   https://ast-grep.github.io/guide/quick-start.html
```

## When to Use

- Finding usage patterns across codebase
- Identifying code structure and organization
- Locating function/class/component definitions
- Analyzing import/dependency patterns
- Refactoring impact analysis
- Code navigation in unfamiliar codebases
