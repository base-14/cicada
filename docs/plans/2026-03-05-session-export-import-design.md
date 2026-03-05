# Session Export/Import — Design

## Overview

Add the ability to export and import Claude Code sessions in Cicada's sessions tab. This enables sharing sessions between machines and users. Exported sessions are fully Claude Code-compatible — recipients can browse them in Cicada and resume them with `claude --resume`.

## Keybindings

| Key | Action |
|-----|--------|
| `e` | Export selected session |
| `E` | Export all sessions (or all filtered sessions) |
| `i` | Import session(s) from zip |

## Export Format

A standard `.zip` file containing:

- `manifest.json` — metadata about the export
- JSONL session file(s) — raw, untouched copies from `~/.claude/projects/`

### Single Session Export (`e`)

**Zip structure:**
```
manifest.json
{uuid}.jsonl
```

**Manifest:**
```json
{
  "version": 1,
  "type": "single",
  "project_path": "-Users-r-work-myproject",
  "session_uuid": "abc-123-def",
  "slug": "happy-cat",
  "exported_at": "2026-03-05T10:00:00Z"
}
```

**Output filename:** `{slug}-{uuid-first-8-chars}.zip` in the current working directory.

**Status bar:** `"Exported to happy-cat-abc12345.zip"`

### Bulk Export (`E`)

Exports all sessions visible in the current view (respects active filter).

**Zip structure:**
```
manifest.json
-Users-r-work-projA/uuid1.jsonl
-Users-r-work-projA/uuid2.jsonl
-Users-r-work-projB/uuid3.jsonl
```

**Manifest:**
```json
{
  "version": 1,
  "type": "bulk",
  "sessions": [
    {"project_path": "-Users-r-work-projA", "session_uuid": "uuid1", "slug": "happy-cat"},
    {"project_path": "-Users-r-work-projA", "session_uuid": "uuid2", "slug": "cool-dog"},
    {"project_path": "-Users-r-work-projB", "session_uuid": "uuid3", "slug": "fast-fox"}
  ],
  "exported_at": "2026-03-05T10:00:00Z"
}
```

**Output filename:** `cicada-export-2026-03-05.zip` in the current working directory.

**Status bar:** `"Exported 42 sessions to cicada-export-2026-03-05.zip"`

## Import Flow (`i`)

1. User presses `i` on the sessions tab.
2. A text input appears asking for the zip file path (supports `~` expansion).
3. Cicada reads the zip and extracts the manifest.
4. **Single session (`type: "single"`):**
   - Shows a project picker listing existing projects from `~/.claude/projects/`.
   - Highlights the original project path if it exists locally.
   - Allows typing a custom project path at the bottom.
   - User selects a project; Cicada copies the JSONL into that project directory.
5. **Bulk import (`type: "bulk"`):**
   - Uses original project paths directly (machine migration use case).
   - Creates project directories if they don't exist.
   - Status bar: `"Imported 42 sessions across 5 projects"`
6. After placing files, adds sessions to the in-memory store (no full re-scan needed).

## New Package: `internal/transfer/`

Handles export and import logic, separate from parser/store/tui.

- `manifest.go` — Manifest types and JSON serialization
- `export.go` — `ExportSession(claudeDir, meta, outputDir) (string, error)` and `ExportAll(claudeDir, metas, outputDir) (string, error)`
- `import.go` — `ReadBundle(zipPath) (*Manifest, map[string][]byte, error)` and `PlaceSession(claudeDir, projectPath, uuid string, data []byte) error`

## UI Components

### Import Path Input

Simple text input overlay (similar to the existing filter input) for entering the zip file path.

### Project Picker (single import only)

Scrollable list of existing project paths with:
- Arrow keys to navigate
- Enter to select
- Bottom entry for typing a custom path
- Original project path highlighted if it exists

## Error Handling

All errors surface in the status bar — no modal dialogs.

- Export: file not found, write permission errors
- Import: invalid zip, missing manifest, corrupt JSONL, target directory not writable

## Future Work

- **Git-friendly export format**: Export JSONL + sidecar `.manifest.json` as plain text files (no zip) for committing sessions to git repos. Text-based, diffable, no repo bloat.

## Testing

- `internal/transfer/` — unit tests for:
  - Export/import round-trip (single and bulk)
  - Manifest parsing and serialization
  - Edge cases: missing fields, bad zip, empty sessions
  - File placement under correct project paths
- Integration: export a session, import to a different project path, verify file lands correctly
