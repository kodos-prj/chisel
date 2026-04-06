# Packmgr to Chisel - Rebranding Documentation Index

## Overview

This directory contains comprehensive rebranding documentation for renaming the project from **Packmgr** to **Chisel**. The analysis identifies all references that need to be renamed and provides a systematic execution plan.

## Documentation Files

### 1. **REBRANDING-QUICK-SUMMARY.txt** (START HERE)
**Best For:** Step-by-step execution
- Organized by phase with exact line numbers
- Quick reference checklist format
- Includes verification commands
- Shows estimated effort: 2-4 hours
- **File size:** 8.5 KB

**When to use:** During actual execution - follow the phases in order

---

### 2. **REBRANDING-ANALYSIS.md** (COMPREHENSIVE REFERENCE)
**Best For:** Detailed planning and understanding
- Complete technical analysis with 10 sections
- All files organized by type and priority
- Risk assessment and mitigation strategies
- Appendices with search patterns and mappings
- **File size:** 23 KB

**When to use:** For planning, understanding impact, and reference during execution

---

### 3. **REBRANDING-FILE-INVENTORY.csv** (TRACKING PROGRESS)
**Best For:** Progress tracking and verification
- Spreadsheet format (open in Excel/Google Sheets)
- One row per file with all metadata
- Specific line numbers and change types
- Can be imported to tracking tools
- **File size:** 5 KB

**When to use:** During execution to track which files have been completed

---

## Quick Facts

| Metric | Value |
|--------|-------|
| Total Files Affected | 25 |
| Total References | ~345 |
| Critical Items | 3 (go.mod, cmd/packmgr/, .gitignore) |
| Estimated Time | 2-4 hours |
| Go Source Files | 10 files, 71 references |
| Test Files | 5 files, 25 references |
| Documentation Files | 8 files, 247 references |
| Config Files | 2 files, 2 references |

## Recommended Workflow

### For Quick Execution
1. Read: **REBRANDING-QUICK-SUMMARY.txt**
2. Follow: Phase 1-8 checklist
3. Verify: Using provided grep commands

### For Comprehensive Understanding
1. Read: **REBRANDING-ANALYSIS.md** (sections 1-3)
2. Review: Summary table in section 6
3. Execute: Using Quick Summary
4. Reference: Appendices as needed

### For Detailed Tracking
1. Open: **REBRANDING-FILE-INVENTORY.csv**
2. Import to: Excel, Google Sheets, or issue tracker
3. Check off: Files as you complete them
4. Reference: Specific lines and change types

---

## Key Changes at a Glance

### Configuration Path Change (Breaking Change)
```
OLD: /etc/packmgr/config.json
NEW: /etc/chisel/config.json
```

### Environment Variables
```
PACKMGR_CONFIG      → CHISEL_CONFIG
PACKMGR_BASE_DIR    → CHISEL_BASE_DIR
PACKMGR_SYMLINK_DIR → CHISEL_SYMLINK_DIR
```

### Module Name
```
module github.com/yourusername/packmgr-go
module github.com/yourusername/chisel-go
```

### Directory Structure
```
cmd/packmgr/ → cmd/chisel/
packmgr (binary) → chisel
```

---

## Critical Files (Do These First)

1. **go.mod** (1 change)
   - Line 1: Module name
   - Impact: Auto-updates all 40+ imports

2. **cmd/packmgr/** (Directory rename)
   - Rename to: cmd/chisel/

3. **.gitignore** (1 change)
   - Line 2: Binary name

---

## High Priority Files (1st Round)

1. **cmd/packmgr/main.go** (22 changes)
2. **pkg/config/config.go** (5 changes)
3. **README.md** (27 changes)
4. **00-SPECIFICATION.md** (94 changes) ← Largest file
5. **docs/CONFIGURATION.md** (45 changes)

---

## Medium Priority Files (2nd Round)

1. **internal/cli/*.go** (7 files, 42 changes)
2. **01-DIAGRAMS.md** (13 changes)
3. **02-CRITICAL-DECISIONS.md** (17 changes)
4. **03-IMPLEMENTATION-PLAN.md** (27 changes)
5. **pkg/config/config_test.go** (5 changes)

---

## Low Priority Files (3rd Round)

1. **Test fixture names** (17 changes)
2. **PHASE2-SUMMARY.md** (22 changes)
3. **PHASE3-UPDATE.md** (2 changes)

---

## Verification Checklist

After completing all changes, verify with these commands:

```bash
# Should return NO results:
rg '\bpackmgr\b' --type go
rg '\bPackmgr\b'
rg '/etc/packmgr/'
rg '\bPACKMGR_'

# Should return results:
rg '\bchisel\b' --type go
rg '/etc/chisel/'
rg '\bCHISEL_'
```

---

## Special Considerations

1. **Breaking Change:** Configuration path changes from `/etc/packmgr/` to `/etc/chisel/`
   - Users will need to migrate config files
   - Create migration guide for users

2. **Module Name:** Changing go module name requires IDE attention
   - Most IDEs will auto-update imports
   - Run `go mod tidy` if needed

3. **Binary Name:** Executable changes from `packmgr` to `chisel`
   - Update any build scripts
   - Update installation instructions

4. **Environment Variables:** Three env vars change
   - Scripts using old names will break
   - Document the change clearly

---

## Git Workflow

```bash
# Create branch for rebranding
git checkout -b rebranding/packmgr-to-chisel

# After all changes:
git add .
git commit -m "refactor: rename packmgr to chisel

- Update module name in go.mod
- Rename cmd/packmgr/ to cmd/chisel/
- Update all imports automatically
- Change config path to /etc/chisel/
- Update environment variables (PACKMGR_* -> CHISEL_*)
- Update all documentation and comments
- Update test fixtures

Affected: 25 files, ~345 references
Breaking: Configuration path change"

git push origin rebranding/packmgr-to-chisel
```

---

## Files Modified by This Analysis

No source files were modified by this analysis. The following reference documents were CREATED:

- `REBRANDING-INDEX.md` (this file)
- `REBRANDING-ANALYSIS.md` (comprehensive analysis)
- `REBRANDING-QUICK-SUMMARY.txt` (execution checklist)
- `REBRANDING-FILE-INVENTORY.csv` (spreadsheet tracking)

---

## Questions & Troubleshooting

### Q: Where do I start?
**A:** Read **REBRANDING-QUICK-SUMMARY.txt** and follow Phase 1-8

### Q: What's the most critical change?
**A:** Updating go.mod first - this triggers import updates across all files

### Q: How long will this take?
**A:** 2-4 hours depending on how carefully you update documentation

### Q: What's the biggest file to update?
**A:** 00-SPECIFICATION.md with 94 changes

### Q: Are there breaking changes?
**A:** Yes - configuration path changes from `/etc/packmgr/` to `/etc/chisel/`

### Q: What if I miss some references?
**A:** Use the verification commands at the end to find any remaining references

---

## Support Files

For detailed information about specific changes:

- **Line-by-line changes:** See REBRANDING-ANALYSIS.md sections 1-5
- **File-by-file breakdown:** See REBRANDING-QUICK-SUMMARY.txt or CSV file
- **Execution order:** See REBRANDING-QUICK-SUMMARY.txt section "RECOMMENDED EXECUTION ORDER"
- **Risk assessment:** See REBRANDING-ANALYSIS.md section 8

---

## Document Versions

- **Analysis Date:** March 24, 2026
- **Total Files Analyzed:** 25
- **Total References Found:** ~345
- **Analysis Complete:** Yes
- **Ready for Execution:** Yes

---

**Start with REBRANDING-QUICK-SUMMARY.txt for immediate execution.**

All reference documents are in the same directory as this file.
