# sy - SiYuan CLI Tool

A command-line interface for SiYuan Note, designed for power users and LLM integration.

## Overview

`sy` is a portable Go CLI tool that provides direct access to SiYuan notes via the kernel API. It enables:
- Quick document listing, searching, and SQL queries
- Export/import of documents in kramdown format (preserving block IDs)
- Interactive editing via `$EDITOR`
- Document management (create, move, delete)

Primary use case: Give LLMs and automation tools easy access to SiYuan notes without needing browser/plugin context.

## Installation

```bash
# From source
go install github.com/xtruder/sy@latest

# Or download binary from releases
```

## Configuration

### API Connection

Configuration is resolved in order: CLI flag > environment variable > config file > default

| Source | Host | Token |
|--------|------|-------|
| Flag | `--host` | `--token` |
| Env | `SIYUAN_HOST` | `SIYUAN_TOKEN` |
| Config | `host:` | `token:` |
| Default | `http://localhost:6806` | (none) |

Config file location: `~/.config/sy/config.yaml`

```yaml
host: http://localhost:6806
token: your-api-token
```

## Commands

### ls - List Documents

List documents in workspace or specific path.

```bash
sy ls                           # List all docs (recursive)
sy ls /Notebook                 # List docs in notebook (non-recursive)  
sy ls /Notebook/Folder          # List docs in folder (non-recursive)
sy ls -r /Notebook              # Force recursive listing
sy ls --sort updated            # Sort by: updated, created, name
sy ls --filter "name:daily"     # Filter documents
```

**Output columns:** ID | Title (truncated) | Path | Updated (human-readable)

**Behavior:**
- Without path argument: recursive listing of entire workspace
- With path argument: non-recursive listing of that path only
- Use `-r` flag to force recursive when path is specified

### export - Export Documents

Export documents as kramdown (preserves block IDs via IAL).

```bash
sy export <doc>                 # Export to stdout
sy export <doc> -o file.md      # Export to file
sy export /Notebook/Folder      # Export all docs in folder
sy export '/Notebook/**'        # Export with glob pattern
sy export -o ./output/ /Notebook  # Export folder to directory
```

**Document identification** (supported for all commands):
- Block ID: `20260118141851-qvxnjey`
- Path: `/Notebook/Folder/Document`
- Human path: `Notebook/Folder/Document`

**Output format** (kramdown with frontmatter):
```markdown
---
siyuan_doc_id: 20260118141851-qvxnjey
siyuan_notebook: 20250501141836-88mugvq
siyuan_path: /Folder/Document
siyuan_hpath: /Notebook/Folder/Document
siyuan_updated: 20260124210821
---
# Document Title
{: id="20260118141851-qvxnjey" updated="20260124210821"}

Paragraph content here.
{: id="20260118141900-abc1234"}

- List item one
  {: id="20260118141910-def5678"}
- List item two
  {: id="20260118141920-ghi9012"}
```

### import - Import Documents

Import markdown/kramdown files into SiYuan.

```bash
sy import file.md                       # Import using frontmatter path
sy import file.md --to /Notebook/Path   # Import to specific path
sy import ./folder/                     # Import all .md files in folder
sy import --title "Doc" --content "Text" --to /Notebook/Path
cat file.md | sy import --to /Notebook/Path  # Import from stdin
```

**Behavior:**
- Reads `siyuan_path` and `siyuan_notebook` from frontmatter if present
- Falls back to `--to` flag if frontmatter missing
- Strips H1 title heading before import (uses title from frontmatter or `--title`)
- Preserves block IDs from kramdown IAL (valid format: `YYYYMMDDHHmmss-7chars`)
- Creates new IDs for blocks without IAL

### edit - Edit Documents

Interactive editing or inline modifications.

```bash
# Interactive editing (opens $EDITOR)
sy edit <doc>                           # Export -> $EDITOR -> Import

# Inline modifications
sy edit <doc> --append "New paragraph at end"
sy edit <doc> --prepend "Paragraph at start"  
sy edit <doc> --after <block-id> "Content after specific block"
sy edit <doc> --replace <block-id> "Replace block content"
sy edit <doc> --delete <block-id>       # Delete specific block
```

**Interactive mode:**
1. Exports document to temp file as kramdown
2. Opens `$EDITOR` (or `--editor` flag)
3. On save, imports modified content back to SiYuan
4. Preserves block IDs from IAL

### new - Create Document

Create a new document.

```bash
sy new /Notebook/Path/Title
sy new /Notebook/Path/Title --content "Initial content"
sy new /Notebook/Path/Title --content -   # Read content from stdin
```

### rm - Delete Document

Delete documents.

```bash
sy rm <doc>                     # Delete (prompts for confirmation)
sy rm <doc> --force             # Skip confirmation
sy rm <doc> --recursive         # Delete doc and all children
sy rm doc1 doc2 doc3            # Delete multiple docs
```

**Behavior:**
- Refuses to delete documents with children unless `--recursive` is passed
- Prompts for confirmation unless `--force` is passed

### mv - Move Document

Move or rename documents.

```bash
sy mv <source> <dest>           # Move to new path
sy mv <doc> --rename "New Title"  # Rename only (keep same location)
```

### search - Search Documents

Full-text search across documents.

```bash
sy search "keyword"             # Search in all docs
sy search "keyword" /Notebook   # Search in specific notebook
sy search --type heading "API"  # Search only in headings
sy search --type paragraph "TODO"  # Search only in paragraphs
```

### sql - Execute SQL Query

Run SQL against SiYuan's block database.

```bash
sy sql "SELECT * FROM blocks WHERE content LIKE '%search%' LIMIT 10"
sy sql --format table "..."     # ASCII table (default)
sy sql --format tsv "..."       # Tab-separated values
sy sql --format json "..."      # JSON array
```

**Available tables:** `blocks`, `spans`, `assets`, `refs`, `file_annotation_refs`

## Kramdown Format

We use SiYuan's native kramdown format which preserves block IDs via Inline Attribute Lists (IAL):

```markdown
Paragraph text.
{: id="20260118141851-abc1234" updated="20260124210821"}

## Heading
{: id="20260118141900-def5678"}

- List item
  {: id="20260118141910-ghi9012"}
```

**Why kramdown:**
- Block references (`((block-id "anchor text"))`) survive round-trips
- Edits preserve existing block IDs
- New content gets new IDs automatically on import
- Native SiYuan format - no conversion loss

**IAL Handling:**
- Export: `getBlockKramdown` API returns content with IAL
- Import: `updateBlock` API preserves valid IDs from IAL
- Valid ID format: `YYYYMMDDHHmmss-xxxxxxx` (14 digits + hyphen + 7 alphanumeric)

## Project Structure

```
sy/
├── cmd/
│   └── sy/
│       └── main.go             # Entry point, Cobra root command
├── internal/
│   ├── api/                    # SiYuan API client
│   │   ├── client.go           # HTTP client, auth, base URL
│   │   ├── notebook.go         # Notebook operations
│   │   ├── document.go         # Document operations  
│   │   ├── block.go            # Block operations (kramdown, update, insert)
│   │   └── query.go            # SQL and search queries
│   ├── cmd/                    # Command implementations
│   │   ├── root.go             # Root command, global flags
│   │   ├── ls.go
│   │   ├── export.go
│   │   ├── import.go
│   │   ├── edit.go
│   │   ├── new.go
│   │   ├── rm.go
│   │   ├── mv.go
│   │   ├── search.go
│   │   └── sql.go
│   ├── config/                 # Configuration loading
│   │   └── config.go
│   ├── format/                 # Output formatting
│   │   ├── table.go
│   │   ├── tsv.go
│   │   └── json.go
│   └── util/                   # Utilities
│       ├── path.go             # Path resolution (ID, path, hpath)
│       ├── frontmatter.go      # YAML frontmatter handling
│       └── editor.go           # $EDITOR integration
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Dependencies

### Core
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management
- `gopkg.in/yaml.v3` - YAML parsing for frontmatter

### Kramdown/Markdown
- `github.com/88250/lute` - SiYuan's markdown/kramdown parser (Go bindings)
  - Handles IAL parsing and serialization
  - Native SiYuan format support

### Output
- `github.com/olekukonko/tablewriter` - ASCII table output
- Standard library for JSON

## API Endpoints Used

| Endpoint | Purpose |
|----------|---------|
| `/api/notebook/lsNotebooks` | List all notebooks |
| `/api/filetree/listDocsByPath` | List documents in path |
| `/api/block/getChildBlocks` | Get top-level children of document |
| `/api/block/getBlockKramdown` | Export block as kramdown with IAL |
| `/api/block/updateBlock` | Update block content (preserves child IAL IDs) |
| `/api/block/insertBlock` | Insert new block |
| `/api/block/deleteBlock` | Delete block |
| `/api/filetree/createDocWithMd` | Create new document |
| `/api/filetree/removeDoc` | Delete document |
| `/api/filetree/moveDocs` | Move documents |
| `/api/filetree/renameDoc` | Rename document |
| `/api/query/sql` | Execute SQL query |
| `/api/search/fullTextSearchBlock` | Full-text search |
| `/api/filetree/getHPathByID` | Get human-readable path |
| `/api/filetree/getHPathByPath` | Get human-readable path from storage path |

## Export Strategy

To export a full document as kramdown with all block IDs preserved:

1. Call `getChildBlocks(docId)` to get top-level block IDs
2. For each top-level block, call `getBlockKramdown(blockId)`
   - Container blocks (lists, blockquotes) include all nested children with IAL
3. Concatenate kramdown results
4. Prepend YAML frontmatter with document metadata

This approach:
- Preserves all block IDs via IAL
- Includes nested content automatically
- Uses N+1 API calls (1 for children list + N for each top-level block)

## Import Strategy

To import kramdown back to SiYuan while preserving block IDs:

1. Parse frontmatter to get `siyuan_doc_id` and path
2. Strip frontmatter from content
3. Strip H1 title heading (title is set via API, not content)
4. Call `updateBlock(docId, kramdownContent)`
   - Document block ID is preserved (API parameter)
   - Child block IDs are preserved from IAL (if valid format)
   - Invalid/missing IDs get new ones generated

## Implementation Phases

### Phase 1: Core Infrastructure
- [ ] Project scaffolding (Go modules, Cobra CLI)
- [ ] Configuration loading (flags > env > config > default)
- [ ] API client with authentication
- [ ] Basic error handling and user-friendly messages
- [ ] Path resolution utilities (ID, path, hpath)

### Phase 2: Read Operations
- [ ] `ls` command - list documents with formatting
- [ ] `export` command - single doc and batch export
- [ ] `sql` command - execute queries with format options
- [ ] `search` command - full-text search

### Phase 3: Write Operations  
- [ ] `new` command - create documents
- [ ] `import` command - import kramdown/markdown
- [ ] `edit` command - interactive and inline editing
- [ ] `rm` command - delete with safety checks
- [ ] `mv` command - move and rename

### Phase 4: Polish
- [ ] Glob pattern support for batch operations
- [ ] Shell completion scripts (bash, zsh, fish)
- [ ] Comprehensive error messages
- [ ] Documentation and examples
- [ ] Release binaries for major platforms

## Non-Goals (v1)

- Watch mode / auto-sync (manual operations only)
- Two-way sync with filesystem mirror
- Template support for `new` command
- Plugin/extension system
- GUI or TUI interface
