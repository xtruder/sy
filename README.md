# sy - SiYuan CLI

[![Go Version](https://img.shields.io/github/go-mod/go-version/xtruder/sy)](https://go.dev/)
[![License](https://img.shields.io/github/license/xtruder/sy)](LICENSE)
[![Release](https://img.shields.io/github/v/release/xtruder/sy)](https://github.com/xtruder/sy/releases)
[![CI](https://github.com/xtruder/sy/actions/workflows/ci.yml/badge.svg)](https://github.com/xtruder/sy/actions/workflows/ci.yml)

A command-line interface for [SiYuan Note](https://b3log.org/siyuan/), designed for power users and LLM integration.

## Usage

### Notebook Management

```bash
sy nb                    # Show current notebook
sy nb Personal           # Set default notebook
sy -n Work ls            # Use specific notebook for command
```

### List Documents

```bash
sy ls                    # List docs at notebook root
sy ls /Folder            # List docs in folder
sy ls -r                 # List all docs recursively
sy ls /Folder -r         # List folder recursively
```

### Export Documents

```bash
sy export /Folder/Doc              # Export as kramdown (preserves block IDs)
sy cat /Folder/Doc                 # Alias for export
sy cat /Folder/Doc --md            # Export as pure markdown
sy export /Doc -o document.md      # Export to file
sy export /Doc --no-frontmatter    # Without YAML frontmatter
```

### Import Documents

```bash
sy import file.md                  # Update doc (uses frontmatter doc_id)
sy import file.md --to /Folder     # Import to specific folder
sy import ./folder/                # Import all .md files
```

### Create Documents

```bash
sy new /Folder/Title               # Create empty document
sy new /Doc --content "# Hello"    # Create with content
echo "Content" | sy new /Doc --content -  # From stdin
```

### Edit Documents

```bash
# Interactive editing
sy edit /Doc                       # Open in $EDITOR
sy edit /Doc --md                  # Edit as pure markdown (new block IDs)

# Inline modifications
sy edit /Doc --append --content "New paragraph"
sy edit /Doc --prepend --content "First paragraph"
sy edit /Doc --after <blockID> --content "After this block"
sy edit /Doc --replace <blockID> --content "Replacement"
sy edit /Doc --delete <blockID>

# Import code files
sy edit /Doc --append --code main.go        # Auto-detect language
sy edit /Doc --append --code script --lang python
```

### Move & Delete

```bash
sy mv /OldPath /NewPath            # Move document
sy mv /Doc --rename "New Title"    # Rename only
sy rm /Doc                         # Delete (with confirmation)
sy rm /Doc --force                 # Delete without confirmation
```

### Search & Query

```bash
sy search "keyword"                # Full-text search
sy sql "SELECT * FROM blocks WHERE content LIKE '%search%' LIMIT 10"
```

### Block References

Block references `((block-id))` are automatically expanded with titles:

```bash
sy edit /Doc --content "See ((20251107131636-pa13zyp))"
# Becomes: See ((20251107131636-pa13zyp "Models"))
```

## Installation

### From Binary

Download the latest release from the [releases page](https://github.com/xtruder/sy/releases).

```bash
# Linux (amd64)
curl -LO https://github.com/xtruder/sy/releases/latest/download/sy-linux-amd64
chmod +x sy-linux-amd64
sudo mv sy-linux-amd64 /usr/local/bin/sy

# macOS (arm64)
curl -LO https://github.com/xtruder/sy/releases/latest/download/sy-darwin-arm64
chmod +x sy-darwin-arm64
sudo mv sy-darwin-arm64 /usr/local/bin/sy

# Windows (amd64)
# Download sy-windows-amd64.exe from releases
```

### From Source

```bash
go install github.com/xtruder/sy/cmd/sy@latest
```

### Shell Completions

```bash
# Bash
sy completion bash > /etc/bash_completion.d/sy

# Zsh
sy completion zsh > "${fpath[1]}/_sy"

# Fish
sy completion fish > ~/.config/fish/completions/sy.fish
```

## Configuration

Configuration is resolved in order: CLI flag > environment variable > config file > default.

| Source | Host | Token | Notebook |
|--------|------|-------|----------|
| Flag | `--host` | `--token` | `-n, --notebook` |
| Env | `SIYUAN_HOST` | `SIYUAN_TOKEN` | `SIYUAN_NOTEBOOK` |
| Config | `host:` | `token:` | `notebook:` |
| Default | `http://localhost:6806` | (none) | (first notebook) |

Config file location: `~/.config/sy/config.yaml`

```yaml
host: http://localhost:6806
token: your-api-token
notebook: Personal
```

## Features

- **Document Management** - List, create, move, rename, and delete documents
- **Export/Import** - Kramdown format with block ID preservation, or pure markdown
- **Interactive Editing** - Edit documents with your favorite `$EDITOR`
- **Inline Modifications** - Append, prepend, replace, or delete blocks
- **Code Import** - Import source files as syntax-highlighted code blocks
- **SQL Queries** - Query SiYuan's block database directly
- **Full-text Search** - Search across all documents
- **Shell Completions** - Tab completion for paths, notebooks, and commands
- **Block References** - Auto-expand `((block-id))` with document titles

## Kramdown Format

SiYuan uses kramdown with Inline Attribute Lists (IAL) to preserve block IDs:

```markdown
---
siyuan_doc_id: 20260118141851-qvxnjey
siyuan_notebook: 20250501141836-88mugvq
siyuan_hpath: /Notebook/Folder/Document
title: Document Title
---

Paragraph content here.
{: id="20260118141900-abc1234" updated="20260124210821"}

## Heading
{: id="20260118141910-def5678"}
```

- **Export**: Block IDs preserved in IAL
- **Import**: Existing IDs preserved, new blocks get new IDs
- **Pure markdown** (`--md`): IAL stripped, all blocks get new IDs

## Development

### Prerequisites

- Go 1.23+
- SiYuan running locally (default: `http://localhost:6806`)

### Build

```bash
git clone https://github.com/xtruder/sy.git
cd sy
make build
```

### Commands

```bash
make build         # Build binary
make test          # Run tests
make lint          # Run linter
make check         # Run all checks (fmt, lint, test, build)
make release       # Build for all platforms
make help          # Show all targets
```

### Project Structure

```
sy/
├── cmd/sy/main.go           # Entry point
├── internal/
│   ├── api/                 # SiYuan API client
│   │   ├── client.go        # HTTP client
│   │   ├── notebook.go      # Notebook operations
│   │   ├── document.go      # Document CRUD
│   │   ├── block.go         # Block operations
│   │   └── query.go         # SQL and search
│   ├── cmd/                 # Command implementations
│   ├── config/              # Configuration
│   ├── format/              # Output formatting
│   └── util/                # Utilities
├── go.mod
├── Makefile
└── README.md
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Related

- [SiYuan](https://github.com/siyuan-note/siyuan) - The note-taking app
- [SiYuan API Documentation](https://github.com/siyuan-note/siyuan/blob/master/API.md)
