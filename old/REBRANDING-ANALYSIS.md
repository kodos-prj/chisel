# Packmgr to Chisel - Rebranding Analysis
## Complete Reference Inventory for Systematic Renaming

**Project:** `/home/abuss/Work/devel/packmgr-go`  
**Date:** March 24, 2026  
**Scope:** All references to "packmgr", "Packmgr", and "/etc/packmgr/"

---

## EXECUTIVE SUMMARY

**Total Files with References:** 24 files  
**Total Reference Categories:**
- Module imports: ~40+ occurrences
- Directory names: 1 (cmd/packmgr)
- Configuration paths: ~15 occurrences of /etc/packmgr/
- Code comments: ~10 occurrences  
- Hardcoded strings in output: ~50+ occurrences
- Test fixtures/temp dirs: ~20 occurrences
- Documentation: ~300+ occurrences

---

## SECTION 1: CRITICAL FILES (MUST RENAME)

### 1.1 GO.MOD - Module Definition
**File Path:** `/home/abuss/Work/devel/packmgr-go/go.mod`  
**Type:** Module configuration  
**Priority:** CRITICAL (First change required)  
**References:**
- Line 1: `module github.com/yourusername/packmgr-go`

**Change Required:**
```
FROM: module github.com/yourusername/packmgr-go
TO:   module github.com/yourusername/chisel-go
```

**Impact:** 
- This change will invalidate ALL imports across the codebase
- Must be done FIRST, before any code changes
- All 40+ import statements will be updated automatically by IDE

---

### 1.2 Directory Structure - CLI Entry Point
**Current Path:** `/home/abuss/Work/devel/packmgr-go/cmd/packmgr/`  
**Type:** Directory name + executable  
**Priority:** CRITICAL

**Changes Required:**
1. Rename directory: `cmd/packmgr/` → `cmd/chisel/`
2. Rename executable (compiled binary): `packmgr` → `chisel`
3. Update .gitignore line 2

**Files in Directory:**
- `main.go` - Contains all command handlers and help text

---

### 1.3 Configuration Path - Default Location
**File Path:** `/home/abuss/Work/devel/packmgr-go/pkg/config/config.go`  
**Type:** Go source with hardcoded paths  
**Priority:** CRITICAL

**Specific Changes:**
```go
Line 13:  DefaultConfigPath = "/etc/packmgr/config.json"
          → DefaultConfigPath = "/etc/chisel/config.json"

Line 15:  DefaultBaseDir is the default base directory for all packmgr data
          → DefaultBaseDir is the default base directory for all chisel data

Line 22:  Config represents the packmgr configuration.
          → Config represents the chisel configuration.

Line 24:  BaseDir is the base directory for all packmgr data (/kod by default)
          → BaseDir is the base directory for all chisel data (/kod by default)
```

**Impact:**
- This is the single source of truth for default config paths
- Change affects environment variables (PACKMGR_* → CHISEL_*)
- Users' existing config files will need migration instructions

---

## SECTION 2: GO SOURCE FILES (Production Code)

### 2.1 cmd/packmgr/main.go
**Path:** `/home/abuss/Work/devel/packmgr-go/cmd/packmgr/main.go`  
**Type:** Main CLI entry point  
**Approximate References:** 22  
**Priority:** HIGH

**Change Categories:**

#### A. Module Imports (8 occurrences)
```go
Line 8:   "github.com/yourusername/packmgr-go/internal/cli"
Line 9:   "github.com/yourusername/packmgr-go/pkg/config"
(WILL AUTO-UPDATE with module rename)
```

#### B. Help Text & Usage Strings (14 occurrences)
Lines: 25, 42, 92, 95, 98, 99, 128, 131, 134, 137, 139, 140, 141, 148

Examples:
```go
Line 25:  "Base directory for packmgr data (overrides config)"
Line 42:  fmt.Printf("packmgr version %s\n", version)
Line 92:  fmt.Println("packmgr - Cross-Distribution Package Manager")
Line 95:  fmt.Println("Usage: packmgr [global-options] <command> [options]")
Line 98:  fmt.Println("  -c, --config <path>   Path to configuration file (default: /etc/packmgr/config.json)")
Line 99:  fmt.Println("  --base-dir <path>     Base directory for packmgr data (default: /kod)")
Line 128-137: All usage examples showing "packmgr" commands
Line 139-140: Configuration path and env var documentation
Line 148:  Comment "Default config file (/etc/packmgr/config.json)"
```

#### C. Environment Variables (3 references)
```go
Line 155: os.Getenv("PACKMGR_CONFIG")
Line 175: os.Getenv("PACKMGR_BASE_DIR")
Line 186: os.Getenv("PACKMGR_SYMLINK_DIR")
```

Should become:
```go
os.Getenv("CHISEL_CONFIG")
os.Getenv("CHISEL_BASE_DIR")
os.Getenv("CHISEL_SYMLINK_DIR")
```

---

### 2.2 pkg/config/config.go
**Path:** `/home/abuss/Work/devel/packmgr-go/pkg/config/config.go`  
**Type:** Configuration management  
**Approximate References:** 5  
**Priority:** HIGH

**Changes:**
- Line 1: Comment "Package config manages packmgr configuration."
- Lines 13, 15, 22, 24: (Already listed in 1.3 above)

---

### 2.3 internal/cli/download.go
**Path:** `/home/abuss/Work/devel/packmgr-go/internal/cli/download.go`  
**Type:** Command implementation  
**Approximate References:** 7  
**Priority:** MEDIUM

**Changes:**
```go
Line 7:   "github.com/yourusername/packmgr-go/pkg/alpm"
Line 8:   "github.com/yourusername/packmgr-go/pkg/config"
Line 9:   "github.com/yourusername/packmgr-go/pkg/download"
Line 25:  // Usage: packmgr download [options] <package> [package2] ...
Line 97:  packmgr download [options] <package> [package2] ...
Line 104-106: Usage examples
```

---

### 2.4 internal/cli/remove.go
**Path:** `/home/abuss/Work/devel/packmgr-go/internal/cli/remove.go`  
**Type:** Command implementation  
**Approximate References:** 10  
**Priority:** MEDIUM

**Changes:**
- Line 8-10: Module imports (auto-update)
- Line 41: Comment "Usage: packmgr remove..."
- Lines 60, 189, 197, 200, 203, 206: Usage strings and examples

---

### 2.5 internal/cli/install.go
**Path:** `/home/abuss/Work/devel/packmgr-go/internal/cli/install.go`  
**Type:** Command implementation  
**Approximate References:** 8  
**Priority:** MEDIUM

**Changes:**
- Lines 10-16: Module imports (auto-update)
- Line 50: Comment "Usage: packmgr install..."

---

### 2.6 internal/cli/info.go
**Path:** `/home/abuss/Work/devel/packmgr-go/internal/cli/info.go`  
**Type:** Command implementation  
**Approximate References:** 4  
**Priority:** MEDIUM

**Changes:**
- Line 1: Comment "Package cli implements command-line interface commands for packmgr."
- Line 11: Comment "InfoCommand implements the 'packmgr info' command."
- Lines 7-8: Module imports (auto-update)

---

### 2.7 internal/cli/search.go
**Path:** `/home/abuss/Work/devel/packmgr-go/internal/cli/search.go`  
**Type:** Command implementation  
**Approximate References:** 3  
**Priority:** MEDIUM

**Changes:**
- Line 1: Comment "Package cli implements command-line interface commands for packmgr."
- Line 11: Comment "SearchCommand implements the 'packmgr search' command."
- Lines 7-8: Module imports (auto-update)

---

### 2.8 internal/cli/sync.go
**Path:** `/home/abuss/Work/devel/packmgr-go/internal/cli/sync.go`  
**Type:** Command implementation  
**Approximate References:** 3  
**Priority:** MEDIUM

**Changes:**
- Line 1: Comment "Package cli implements command-line interface commands for packmgr."
- Line 12: Comment "SyncCommand implements the 'packmgr sync' command."
- Lines 8-9: Module imports (auto-update)

---

### 2.9 internal/cli/extract.go
**Path:** `/home/abuss/Work/devel/packmgr-go/internal/cli/extract.go`  
**Type:** Command implementation  
**Approximate References:** 7  
**Priority:** MEDIUM

**Changes:**
- Line 25: Comment "Usage: packmgr extract..."
- Lines 134, 141-143: Usage examples
- Lines 8-9: Module imports (auto-update)

---

### 2.10 pkg/store/store.go
**Path:** `/home/abuss/Work/devel/packmgr-go/pkg/store/store.go`  
**Type:** Storage implementation  
**Approximate References:** 2  
**Priority:** MEDIUM

**Changes:**
- Line 12: Module import (auto-update)

---

## SECTION 3: TEST FILES (Test Code)

### 3.1 internal/cli/remove_test.go
**Path:** `/home/abuss/Work/devel/packmgr-go/internal/cli/remove_test.go`  
**Type:** Unit tests  
**Approximate References:** 3  
**Priority:** MEDIUM

**Changes:**
- Lines 9-11: Module imports (auto-update)

---

### 3.2 pkg/config/config_test.go
**Path:** `/home/abuss/Work/devel/packmgr-go/pkg/config/config_test.go`  
**Type:** Unit tests  
**Approximate References:** 5  
**Priority:** MEDIUM

**Changes:**
- Line 143: JSON test data "/opt/packmgr" → "/opt/chisel"
- Lines 157-158, 162, 167: Assertions checking paths with packmgr

Examples:
```go
Line 143:  "base_dir": "/opt/packmgr"
Line 157:  if cfg.BaseDir != "/opt/packmgr" {
Line 158:  t.Errorf("Expected BaseDir /opt/packmgr, got %s", cfg.BaseDir)
Line 162:  expectedStore := "/opt/packmgr/store"
Line 167:  expectedRegistry := "/opt/packmgr/registry.json"
```

---

### 3.3 pkg/alpm/alpm_test.go
**Path:** `/home/abuss/Work/devel/packmgr-go/pkg/alpm/alpm_test.go`  
**Type:** Unit tests  
**Approximate References:** 5  
**Priority:** LOW (Test fixture names)

**Changes:** (Low priority - test fixture names)
```go
Line 16:   tmpDir, err := os.MkdirTemp("", "packmgr-alpm-test-*")
Line 227:  tmpDir, err := os.MkdirTemp("", "packmgr-check-*")
Line 256:  tmpDir, err := os.MkdirTemp("", "packmgr-alpm-test-*")
Line 490:  tmpDir, err := os.MkdirTemp("", "packmgr-alpm-test-*")
Line 535:  tmpDir, err := os.MkdirTemp("", "packmgr-bench-*")
```

Recommended changes:
```go
→ "chisel-alpm-test-*"
→ "chisel-check-*"
→ "chisel-alpm-test-*"
→ "chisel-alpm-test-*"
→ "chisel-bench-*"
```

---

### 3.4 pkg/alpm/alpm_integration_test.go
**Path:** `/home/abuss/Work/devel/packmgr-go/pkg/alpm/alpm_integration_test.go`  
**Type:** Integration tests  
**Approximate References:** 2  
**Priority:** LOW (Test fixture names)

**Changes:**
```go
Line 23:   tmpDir, err := os.MkdirTemp("", "packmgr-integration-*")
Line 219:  tmpDir, err := os.MkdirTemp("", "packmgr-multi-repo-*")
```

---

### 3.5 pkg/database/sync_test.go
**Path:** `/home/abuss/Work/devel/packmgr-go/pkg/database/sync_test.go`  
**Type:** Integration tests  
**Approximate References:** 10  
**Priority:** LOW (Test fixture names)

**Changes:** (All test fixture directory names)
```go
Lines: 40, 77, 109, 156, 190, 211, 237, 275, 290
All follow pattern: os.MkdirTemp("", "packmgr-test-*")
→ os.MkdirTemp("", "chisel-test-*")
```

---

## SECTION 4: DOCUMENTATION FILES

### 4.1 README.md
**Path:** `/home/abuss/Work/devel/packmgr-go/README.md`  
**Type:** Main documentation  
**Approximate References:** 27  
**Priority:** HIGH

**Change Categories:**

#### A. Title and Introduction (Multiple)
```
Line 1:   # Packmgr - Cross-Distribution Package Manager
Line 3:   Bring **Arch Linux packages to ANY Linux distribution**. Packmgr runs...
Line 5:   Packmgr is a **cross-distribution package manager**...
Line 7:   Packmgr is a **cross-distribution package manager** that solves...
Line 9:   **Without Packmgr:**
Line 15:  **With Packmgr:**
```

#### B. Command Examples (Multiple)
```
Lines 17-18:  packmgr sync, packmgr install python
Lines 25:     Packmgr brings Arch packages...
Lines 120, 123, 126: Build and usage examples with ./packmgr
Lines 134:    ### What is Packmgr?
```

#### C. Configuration Paths (Multiple)
```
Line 252:  **Config**: JSON format (`/etc/packmgr/config.json`)
Line 259:  **Configuration**: `/etc/packmgr/config.json` (JSON, not YAML)
```

#### D. Project Structure (Multiple)
```
Line 417:  └── packmgr/          # Main CLI entry point
```

---

### 4.2 00-SPECIFICATION.md
**Path:** `/home/abuss/Work/devel/packmgr-go/00-SPECIFICATION.md`  
**Type:** Technical specification  
**Approximate References:** 94  
**Priority:** HIGH

**Critical Sections to Update:**
- Line 1: Title "# Packmgr - Symlink-Based Package Manager"
- Line 32: "Packmgr is a **cross-distribution package manager**..."
- Lines 58, 70, 86, 90, 102, 147: Multiple instances
- Line 156: `/kod/` comments mentioning Packmgr
- Lines 191-193: Wrapper comments
- Line 198: "Unlike pacman which updates databases automatically, packmgr uses..."
- Line 202: Command example `packmgr sync`
- Line 214, 259, 266: Multiple command examples
- Line 352: "The **package store** is a centralized directory..."
- Line 403: Comment "Auto-generated by packmgr"
- Line 409: "Clear what's managed by packmgr"
- Lines 444, 482, 512-517, 535, 544-554: Multiple command examples
- Lines 568, 584, 602-605, 617, 633-639, 646-649, 660-665: More examples
- Line 680: Project structure
- Line 806: "CLI receives: packmgr install vim"
- Lines 837, 845, 877: Test examples
- Line 971: Storage warning about packmgr
- Line 979: "Configuration in JSON format at `/etc/packmgr/config.json`"
- Line 1003: Log file path example
- Lines 1134-1274: Complete command reference documentation

---

### 4.3 01-DIAGRAMS.md
**Path:** `/home/abuss/Work/devel/packmgr-go/01-DIAGRAMS.md`  
**Type:** Architecture diagrams  
**Approximate References:** 13  
**Priority:** MEDIUM

**Sections:**
- Lines 72-74: Project structure diagram
- Line 164: "Packmgr runs on ANY distribution"
- Line 187: "Key Principle: Packmgr brings Arch packages..."
- Lines 246, 297, 385, 477, 553, 588: Multiple command examples

---

### 4.4 02-CRITICAL-DECISIONS.md
**Path:** `/home/abuss/Work/devel/packmgr-go/02-CRITICAL-DECISIONS.md`  
**Type:** Design documentation  
**Approximate References:** 17  
**Priority:** MEDIUM

**Sections:**
- Line 3: Title mention
- Line 29: "NEW in v4.0: Need to decide whether packmgr should work..."
- Lines 70, 316, 320, 326, 333, 346, 366-378: Multiple instances
- Lines 675, 924-925, 949, 1001: Design decision explanations

---

### 4.5 03-IMPLEMENTATION-PLAN.md
**Path:** `/home/abuss/Work/devel/packmgr-go/03-IMPLEMENTATION-PLAN.md`  
**Type:** Implementation roadmap  
**Approximate References:** 27  
**Priority:** MEDIUM

**Sections:**
- Line 3: Title mention
- Line 247-249: Command implementation checklist
- Lines 263-267, 271-296: Setup and testing examples
- Line 398: File reference
- Lines 407-410, 706, 812-844: Command documentation

---

### 4.6 PHASE2-SUMMARY.md
**Path:** `/home/abuss/Work/devel/packmgr-go/PHASE2-SUMMARY.md`  
**Type:** Development phase notes  
**Approximate References:** 22  
**Priority:** LOW (Historical)

**Sections:** Multiple command examples and references throughout

---

### 4.7 PHASE3-UPDATE.md
**Path:** `/home/abuss/Work/devel/packmgr-go/PHASE3-UPDATE.md`  
**Type:** Development phase notes  
**Approximate References:** 2  
**Priority:** LOW (Historical)

---

### 4.8 docs/CONFIGURATION.md
**Path:** `/home/abuss/Work/devel/packmgr-go/docs/CONFIGURATION.md`  
**Type:** Configuration guide  
**Approximate References:** 45  
**Priority:** HIGH

**Critical Sections:**

#### A. Configuration Paths
```
Line 65:   **Default location:** `/etc/packmgr/config.json`
Line 89:   Base directory for all packmgr data
Lines 202-212: Examples with `/home/user/.local/packmgr`
```

#### B. Environment Variables
```
Lines 46-51, 54, 59: Environment variable examples
PACKMGR_CONFIG → CHISEL_CONFIG
PACKMGR_BASE_DIR → CHISEL_BASE_DIR
```

#### C. Command Examples (Multiple)
```
Lines 22-32, 50-51, 54, 236-237, 246-250, 259-263, 279-283, 294-295, 316, 351-352, 364-371
All contain "packmgr" commands
```

#### D. Configuration File Examples
```
Lines 308, 316: Import and usage examples
```

---

## SECTION 5: BUILD AND CONFIGURATION FILES

### 5.1 .gitignore
**Path:** `/home/abuss/Work/devel/packmgr-go/.gitignore`  
**Type:** Git configuration  
**References:** 1  
**Priority:** MEDIUM

**Change:**
```
Line 2: packmgr
→ chisel
```

---

### 5.2 go.mod (Already listed in 1.1)

**File Path:** `/home/abuss/Work/devel/packmgr-go/go.mod`  
**Single Reference:**
```
Line 1: module github.com/yourusername/packmgr-go
→ module github.com/yourusername/chisel-go
```

---

## SECTION 6: SUMMARY TABLE BY FILE TYPE

### Go Source Files (Production)
| File | References | Priority | Types |
|------|-----------|----------|-------|
| cmd/packmgr/main.go | 22 | HIGH | Imports, help text, env vars, paths |
| pkg/config/config.go | 5 | HIGH | Comments, hardcoded paths |
| internal/cli/download.go | 7 | MEDIUM | Imports, comments, examples |
| internal/cli/remove.go | 10 | MEDIUM | Imports, comments, examples |
| internal/cli/install.go | 8 | MEDIUM | Imports, comments |
| internal/cli/info.go | 4 | MEDIUM | Imports, comments |
| internal/cli/search.go | 3 | MEDIUM | Imports, comments |
| internal/cli/sync.go | 3 | MEDIUM | Imports, comments |
| internal/cli/extract.go | 7 | MEDIUM | Imports, comments, examples |
| pkg/store/store.go | 2 | MEDIUM | Imports |
| **Subtotal** | **71** | | |

### Test Files
| File | References | Priority | Types |
|------|-----------|----------|-------|
| pkg/config/config_test.go | 5 | MEDIUM | Test paths |
| internal/cli/remove_test.go | 3 | MEDIUM | Imports |
| pkg/alpm/alpm_test.go | 5 | LOW | Temp dir names |
| pkg/alpm/alpm_integration_test.go | 2 | LOW | Temp dir names |
| pkg/database/sync_test.go | 10 | LOW | Temp dir names |
| **Subtotal** | **25** | | |

### Documentation Files
| File | References | Priority | Types |
|------|-----------|----------|-------|
| README.md | 27 | HIGH | Headers, examples, paths |
| 00-SPECIFICATION.md | 94 | HIGH | Comprehensive technical spec |
| docs/CONFIGURATION.md | 45 | HIGH | Configuration guide |
| 01-DIAGRAMS.md | 13 | MEDIUM | Diagrams, examples |
| 02-CRITICAL-DECISIONS.md | 17 | MEDIUM | Design decisions |
| 03-IMPLEMENTATION-PLAN.md | 27 | MEDIUM | Implementation guide |
| PHASE2-SUMMARY.md | 22 | LOW | Phase notes |
| PHASE3-UPDATE.md | 2 | LOW | Phase notes |
| **Subtotal** | **247** | | |

### Configuration Files
| File | References | Priority | Types |
|------|-----------|----------|-------|
| go.mod | 1 | CRITICAL | Module name |
| go.sum | 0 | N/A | (Auto-generated) |
| .gitignore | 1 | MEDIUM | Binary name |
| **Subtotal** | **2** | | |

### Directory Structure
| Item | Priority | Change |
|------|----------|--------|
| cmd/packmgr/ | CRITICAL | → cmd/chisel/ |
| packmgr (binary) | CRITICAL | → chisel |

---

## SECTION 7: EXECUTION PLAN

### Phase 1: Preparation (BEFORE any changes)
1. Backup the entire project
2. Create a new git branch for the rebranding
3. Verify all tests pass before starting

### Phase 2: Critical Changes (FIRST)
**Do these in this order:**

1. **Update go.mod** - Change module name
   - Line 1: `module github.com/yourusername/packmgr-go` → `module github.com/yourusername/chisel-go`

2. **IDE will offer to update all imports** - Accept all

3. **Rename directories**
   - `cmd/packmgr/` → `cmd/chisel/`
   - Update any build scripts that reference this path

4. **Update .gitignore**
   - Line 2: `packmgr` → `chisel`

### Phase 3: Configuration & Environment Variables
1. Update `pkg/config/config.go`:
   - Line 13: `/etc/packmgr/config.json` → `/etc/chisel/config.json`
   - Lines 1, 15, 22, 24: Comments

2. Update all env var references:
   - `PACKMGR_CONFIG` → `CHISEL_CONFIG`
   - `PACKMGR_BASE_DIR` → `CHISEL_BASE_DIR`
   - `PACKMGR_SYMLINK_DIR` → `CHISEL_SYMLINK_DIR`

### Phase 4: Command Handler Strings
Update all CLI help text and usage examples in:
- `cmd/packmgr/main.go` (22 references)
- `internal/cli/download.go` (7 references)
- `internal/cli/remove.go` (10 references)
- `internal/cli/install.go`, `info.go`, `search.go`, `sync.go`, `extract.go`

### Phase 5: Test Files
1. Update test data and assertions in:
   - `pkg/config/config_test.go` - Update test paths from `/opt/packmgr/` to `/opt/chisel/`
   - Other test files - Update temp directory name prefixes

### Phase 6: Documentation
**HIGH PRIORITY:**
- README.md - 27 references
- 00-SPECIFICATION.md - 94 references
- docs/CONFIGURATION.md - 45 references

**MEDIUM PRIORITY:**
- 01-DIAGRAMS.md - 13 references
- 02-CRITICAL-DECISIONS.md - 17 references
- 03-IMPLEMENTATION-PLAN.md - 27 references

**LOW PRIORITY (Historical):**
- PHASE2-SUMMARY.md - 22 references
- PHASE3-UPDATE.md - 2 references

### Phase 7: Build & Test
1. Run: `go build -o chisel ./cmd/chisel`
2. Run full test suite: `go test ./...`
3. Verify help text: `./chisel help`
4. Manual smoke tests for each command

### Phase 8: Final Verification
1. Verify compiled binary works: `./chisel --version`
2. Check all commands show correct help text
3. Verify config paths in help text show `/etc/chisel/`
4. Run entire test suite

---

## SECTION 8: RISK ASSESSMENT

### High Risk Areas
1. **Module imports** - Large number, but IDE can auto-fix
2. **Configuration paths** - Breaking change for existing users
3. **Command examples in docs** - Easy to miss some

### Medium Risk Areas
1. **Environment variable names** - Users may have scripts using old names
2. **Test fixture names** - Non-critical but good for consistency

### Low Risk Areas
1. **Comments and documentation** - Purely informational
2. **Historical phase notes** - Already archived

### Mitigation Strategies
1. Create MIGRATION.md documenting the change
2. Support legacy env var names for backward compatibility (optional)
3. Create symbolic links for old config path (optional)
4. Update GitHub repository references

---

## SECTION 9: FILES REQUIRING NO CHANGES

- `go.sum` - Auto-generated, will update after go.mod change
- All other source files in `pkg/` not listed above
- Internal utility files that don't reference the project name

---

## SECTION 10: NOTES FOR DIFFERENT PLATFORMS

### When running on Ubuntu/Debian
- Users should update config path in startup scripts
- System-wide config would move from `/etc/packmgr/` to `/etc/chisel/`

### When running on Fedora
- Similar config path updates needed

### Docker/Container Changes
- Any Dockerfile references to packmgr should be updated
- No Dockerfile files found in this codebase (checked)

---

## APPENDIX A: QUICK REFERENCE - ALL FILES TO MODIFY

```
CRITICAL (modify first):
1. go.mod
2. cmd/packmgr/ (rename to cmd/chisel/)
3. .gitignore

HIGH PRIORITY:
4. cmd/packmgr/main.go (rename when dir is renamed)
5. pkg/config/config.go
6. README.md
7. 00-SPECIFICATION.md
8. docs/CONFIGURATION.md

MEDIUM PRIORITY:
9. internal/cli/download.go
10. internal/cli/remove.go
11. internal/cli/install.go
12. internal/cli/info.go
13. internal/cli/search.go
14. internal/cli/sync.go
15. internal/cli/extract.go
16. 01-DIAGRAMS.md
17. 02-CRITICAL-DECISIONS.md
18. 03-IMPLEMENTATION-PLAN.md
19. pkg/config/config_test.go

LOW PRIORITY:
20. pkg/alpm/alpm_test.go
21. pkg/alpm/alpm_integration_test.go
22. pkg/database/sync_test.go
23. internal/cli/remove_test.go
24. PHASE2-SUMMARY.md
25. PHASE3-UPDATE.md
```

---

## APPENDIX B: SEARCH PATTERNS FOR VERIFICATION

After completion, verify changes with these regex patterns:

```
# Should find NO results:
\bpackmgr\b (except in comments about the old project)
Packmgr\b
/etc/packmgr/
PACKMGR_

# Should find results only in new locations:
chisel (in filenames, imports, help text)
Chisel (in documentation)
/etc/chisel/
CHISEL_
```

---

## APPENDIX C: ENVIRONMENT VARIABLE MAPPING

| Old Variable | New Variable | Files Using |
|--------------|--------------|-------------|
| PACKMGR_CONFIG | CHISEL_CONFIG | cmd/packmgr/main.go, docs/CONFIGURATION.md |
| PACKMGR_BASE_DIR | CHISEL_BASE_DIR | cmd/packmgr/main.go, docs/CONFIGURATION.md |
| PACKMGR_SYMLINK_DIR | CHISEL_SYMLINK_DIR | cmd/packmgr/main.go |

---

## APPENDIX D: CONFIGURATION PATH MAPPING

| Aspect | Old | New | Files |
|--------|-----|-----|-------|
| Config file path | /etc/packmgr/config.json | /etc/chisel/config.json | pkg/config/config.go, cmd/packmgr/main.go, docs/CONFIGURATION.md |
| Base directory | /kod | /kod | (No change, independent) |
| Store path | /kod/store or /etc/packmgr/... | /kod/store or /etc/chisel/... | Derived from base |

---

END OF ANALYSIS
