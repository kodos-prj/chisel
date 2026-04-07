# Registry.json Analysis: Chisel vs. Kodos Architecture

## Executive Summary

This analysis examines the current usage of `registry.json` in chisel2 and evaluates the feasibility of moving it to `/var/kod` (generation-specific path) following Kodos architecture patterns. The key finding is that while Kodos uses generation-specific paths for metadata storage, chisel's current single-file design has fundamental architectural differences that would require significant refactoring to support generation-based management.

---

## Part 1: Current Registry.json Usage in Chisel

### 1.1 How Registry.json is Currently Used

**File Location & Initialization:**
- Default path: `/kod/registry.json`
- Configurable via `Config.RegistryPath`
- Located in code: `/home/abuss/Work/devel/chisel2/pkg/registry/registry.go:14`

**Data Structure:**
```go
type Package struct {
    Name         string   // Package name
    Version      string   // Version string
    Source       string   // "official" or "aur"
    Repository   string   // e.g., "core", "extra", "aur"
    Files        []string // All extracted files
    Executables  []string // Executables in usr/bin, usr/sbin
    Dependencies []string // Tracked dependencies
    InstallDate  string   // RFC3339 timestamp
    UpdateDate   string   // RFC3339 timestamp (omitted if not set)
}

type Registry struct {
    path     string
    packages map[string]*Package  // In-memory cache
    mu       sync.RWMutex         // Thread-safe access
}
```

**File Format:**
- JSON marshaled map of package names to Package structs
- Pretty-printed with 2-space indentation
- Represents: `map[string]*Package` where key is package name

**I/O Operations:**

| Operation | Method | Details |
|-----------|--------|---------|
| Load | `Load()` - registry.go:57 | Reads entire JSON from disk into memory |
| Save | `Save()` - registry.go:70 | Marshals entire map to JSON and writes to disk |
| Add | `AddPackage()` - registry.go:83 | Adds/updates in-memory map, no immediate disk write |
| Remove | `RemovePackage()` - registry.go:92 | Removes from in-memory map, no immediate disk write |
| Query | `GetPackage()`, `ListPackages()` - registry.go:101-119 | Reads from in-memory map |

### 1.2 Is Registry Read/Written Frequently During Operations?

**YES - Pattern: Read-Once, Modify-Many, Write-Once**

**Command Operation Flow:**

1. **Install Command** (install.go:483-528):
   - Opens registry (1 read from disk)
   - Loops through all packages to install
   - Calls `reg.AddPackage()` for each (memory operations)
   - Calls `reg.Save()` ONCE at end (1 write to disk)
   - Pattern: Read → Cache → Batch Updates → Single Write

2. **Remove Command** (remove.go:72-176):
   - Opens registry (1 read from disk)
   - Checks dependencies against all installed packages
   - Calls `reg.RemovePackage()` for each (memory operations)
   - Calls `reg.Save()` ONCE at end (1 write to disk)
   - Pattern: Read → Modify-In-Memory → Single Write

3. **Upgrade Command** (upgrade.go:102-170):
   - Opens registry (1 read from disk)
   - Reads `reg.ListPackages()` for comparison
   - Re-uses InstallCommand for each upgrade
   - Calls `reg.Save()` inside InstallCommand
   - Pattern: Multiple Read-Modify-Write cycles (problematic!)

4. **List Command** (list.go:29):
   - Opens registry (1 read from disk)
   - Reads `reg.ListPackages()` (memory operation)
   - No writes

5. **Cleanup Command** (cleanup.go:118):
   - Opens registry (1 read from disk)
   - Reads `reg.ListPackages()` to find active versions
   - No writes to registry (only filesystem operations)

**Disk I/O Summary:**
- **Most commands**: 1 disk read at start, 1 disk write at end (efficient)
- **Upgrade command**: Multiple reads/writes across multiple invocations
- **Frequency**: Registry is accessed on every package operation
- **Typical workflow**: Install (1 reg write), Upgrade (N reg writes), List (1 reg read), Remove (1 reg write)

### 1.3 Is Registry Ever Locked or Has Concurrency Issues?

**Current Locking Mechanism:**

**Registry Level** (registry.go:34):
```go
type Registry struct {
    mu sync.RWMutex  // In-process locking only
}
```

- Uses Go's `sync.RWMutex` for thread-safe in-memory access
- Protects against concurrent access within same process
- **Provides NO inter-process locking** (different `chisel` commands in parallel)

**File Level:**
- **NO file-level locking** (os.WriteFile is atomic on most filesystems, but not guaranteed)
- **NO pre-flight checks** before reading (TOCTOU race condition possible)
- **NO exclusive locks** between read and write phases

**Concurrency Issues:**

1. **Multi-User Same System**: If two users run `chisel install` simultaneously:
   - User A: Reads registry, installs pkg1, writes registry
   - User B: Reads registry, installs pkg2, writes registry
   - **Result**: User B's changes lost (B read stale data)

2. **Upgrade Cycles**: If upgrade crashes mid-operation:
   - Some packages may be installed but registry not updated
   - No rollback mechanism
   - Registry state inconsistent with filesystem state

3. **Package Deletion Race**: If symlink removal fails but registry is written:
   - Registry doesn't reflect actual symlink state
   - Next operation may fail unexpectedly

**Current Mitigation**: 
- Single-user assumption (all `chisel` operations run as root or same user)
- In-process mutex prevents concurrent modification within single command
- Design accepts filesystem state divergence from registry state

### 1.4 Does Registry Need to Persist Across Package Manager Invocations?

**YES - Registry is Persistent State**

**Across Invocations:**
- Each `chisel` command invocation reads current registry.json
- Modifications persist to next invocation
- Used to track which packages are installed (prevents re-installing)
- Used to identify orphaned versions during cleanup

**Across System Reboots:**
- Registry persists in `/kod/registry.json`
- Survives system restart
- Essential for `list`, `upgrade`, `cleanup` commands on next boot

**Across Package Manager Upgrades:**
- If chisel binary is updated, old registry.json is still readable
- Version tracking in registry allows incremental updates
- Install date preserved across upgrades

**State Continuity Requirements:**
- Package installation status must survive command boundaries
- Dependency tracking must survive to support remove operations
- File lists must survive for cleanup and removal

---

## Part 2: What Happens If Registry.json is Lost or Corrupted?

### 2.1 Can Packages Still Function?

**YES - Packages Function Independently**

**Functional Components Survived:**
- **Store**: `/kod/store/{package}/{version}/` contains all files
- **Symlinks**: System-level symlinks in `/` still point to store
- **Wrappers**: `/kod/wrappers/{executable}` wrapper scripts still executable
- **Package Files**: All extracted files remain intact in store

**Example After registry.json Deletion:**
- Command execution: WORKS (wrappers still point to store)
- Application functionality: WORKS (all binaries/libraries in place)
- System integrity: WORKS (symlinks still valid)

### 2.2 What Recovery Options Exist?

**Option 1: Rebuild Registry from Store (NOT IMPLEMENTED)**

Current code could theoretically scan the store and reconstruct:
```
For each /kod/store/{package}/{version}/:
  - Create registry entry
  - List all files
  - Identify executables in usr/bin, usr/sbin
  - Estimate install date from filesystem mtime
  - Guess source (official vs aur) - WOULD FAIL
```

**Problem**: Source information (official vs aur) is not stored in filesystem
- Cannot distinguish whether package came from Arch repos or AUR
- Cannot recreate dependencies list (not stored in extracted files)
- Install date accuracy lost (would use current mtime)

**Option 2: Reconstruct from Installed Packages (PARTIAL)**

Could query installed packages in store to find versions:
- List all directories in `/kod/store/`
- Get current versions from symlinks at `/kod/store/{package}/current`
- Does NOT recover file lists, executables, dependencies, dates

**Option 3: Manual Reconstruction (MANUAL)**

User could manually run:
```bash
chisel remove <all-packages>  # Cleans up symlinks, wrappers, registry
chisel install <all-packages> # Reinstalls everything, recreates registry
```

**Option 4: System Restore (SYSTEM-LEVEL)**

- If `/kod` directory has snapshots or backups, restore from backup
- If version control used, restore registry.json from git
- If system has BTRFS snapshots, rollback filesystem state

**Current Status**: NO automatic recovery implemented
- Users must manually rebuild registry or use backups
- Recommendation: Backup registry.json regularly (e.g., cron job)

### 2.3 Current Behavior on Corruption

**Scenario: JSON Parsing Fails**

Code path (registry.go:38-54):
```go
func NewRegistry(path string) (*Registry, error) {
    r := &Registry{...}
    if err := r.Load(); err != nil && !os.IsNotExist(err) {
        return nil, fmt.Errorf("failed to load registry: %w", err)
    }
    return r, nil
}

func (r *Registry) Load() error {
    data, err := os.ReadFile(r.path)
    if err != nil {
        return err
    }
    return json.Unmarshal(data, &r.packages)  // Fails on corruption
}
```

**Behavior:**
- If JSON unparseable: Command returns error
- If file missing: Creates new empty registry (no error)
- If file corrupted: Installation fails with JSON parse error

**User Experience:**
```
$ chisel install vim
Error: failed to load registry: invalid character 'x' looking for beginning of value
```

**Result**: Installation cannot proceed until registry is manually repaired

---

## Part 3: Current Design and Multi-User/Multi-System Scenarios

### 3.1 Multiple Users on Same System

**Current Behavior: UNSUPPORTED**

**Scenario**: Users root and alice both run chisel commands

**Race Condition**: No inter-process locking
```
Time  | root                      | alice
------|---------------------------|---------------------------
T1    | Opens /kod/registry.json  |
T2    |                           | Opens /kod/registry.json
T3    | Installs pkg1             |
T4    | Writes registry.json      |
T5    |                           | Installs pkg2
T6    |                           | Overwrites registry.json (missing pkg1!)
```

**Permissions Issue**: Root vs non-root
- `/kod/registry.json` owned by root (likely)
- Non-root user cannot write registry.json
- Error: "permission denied"

**Design Assumption**: Single administrative user (root) performs all package operations

### 3.2 System Upgrades or Reinstalls

**Scenario 1: OS Upgrade (chisel binary updated)**
- Old registry.json remains in `/kod/registry.json`
- New chisel version reads and modifies old registry
- Backward compatibility: Registry format unchanged in codebase
- Status: WORKS (but no migration path for format changes)

**Scenario 2: Fresh System Install**
- `/kod/` empty at first
- First `chisel install` creates `/kod/registry.json`
- Registry populates as packages installed
- Status: WORKS

**Scenario 3: Filesystem Migration**
- Moving `/kod/` to different partition
- Registry.json moves with it (same path)
- Config needs update if path changes
- Status: Works if path is updated, requires manual intervention

**Scenario 4: System Reinstall with Data Preservation**
- Old `/kod/registry.json` remains from previous install
- New chisel commands read old state
- Potential for stale entries (packages removed externally)
- Status: Works but may have consistency issues

### 3.3 Atomicity of Package Operations

**Current Implementation: NO ATOMICITY GUARANTEES**

**All-or-Nothing Atomicity: NOT PROVIDED**

**Example: Install Fails Mid-Way**

```
Steps in Install:
1. Resolve dependencies ✓
2. Download packages ✓
3. Extract packages ✓
4. Create symlinks ✓
5. Generate wrappers ✓
6. Update registry ✓
7. DELETE package cache (not shown in install.go)
```

If failure at step 7 (cache cleanup fails):
- Packages installed and functional ✓
- Registry updated ✓
- Symlinks in place ✓
- Wrappers generated ✓
- Cache not cleaned ✗
- **Partial success** - cache has stale files but system works

**Example: Symlink Creation Fails**

If failure during symlink creation (step 4):
- Packages extracted ✓
- Some symlinks created ✓
- Symlink creation failed (file conflict) ✗
- Registry updated with ALL packages ✗ (WRONG!)
- Wrappers still generated ✓
- **Inconsistent state**: Registry says packages installed but some symlinks missing

**Code Evidence** (install.go:319-421):
```go
// Create symlinks - failures logged but continue
for _, filePath := range pkgFileInfo.AllFiles {
    if err := os.Symlink(...); err != nil {
        fmt.Fprintf(os.Stderr, "! Warning: Failed to create symlink...\n")
        // CONTINUES LOOP - does not abort
    }
}

// Registry update happens regardless
reg, err := registry.NewRegistry(i.config.RegistryPath)
for _, pkg := range toInstall {
    if err := reg.AddPackage(regPkg); err != nil {
        // Logs warning but CONTINUES
    }
}
if err := reg.Save(); err != nil {
    return fmt.Errorf("failed to save registry: %w", err)
}
```

**Result**: Install proceeds even with symlink failures
- Registry.json written with all packages marked installed
- But symlinks may not exist
- Next remove command expects symlinks to exist (fails)

**Rollback Capability: NO**

If critical step fails:
- No automatic rollback
- Must manually clean up (remove installed packages, delete extracted files)
- No transaction log to understand what succeeded/failed
- No checkpoint-recovery mechanism

**Crash During Write**

If process crashes during `reg.Save()`:
```go
func (r *Registry) Save() error {
    data, err := json.MarshalIndent(r.packages, "", "  ")
    return os.WriteFile(r.path, data, 0644)  // CRASH HERE?
}
```

`os.WriteFile()` is:
- **Atomic on most filesystems** (writes to temp file, renames)
- **NOT atomic on all filesystems** (some do in-place writes)
- **Lost data possible** if system crash during write

**No Transaction Support**:
- No WAL (Write-Ahead Logging)
- No transaction markers
- No way to verify registry integrity

---

## Part 4: Kodos Generation Architecture

### 4.1 What is a "Generation" in Kodos?

**Definition**: A numbered snapshot of the entire system state using Btrfs subvolumes

**Directory Structure**:
```
/kod/generations/
├── 0/                          # Generation 0 (initial)
│   ├── rootfs                  # BTRFS subvolume: /
│   ├── installed_packages      # JSON: packages to install
│   ├── enabled_services        # Text: services enabled
│   └── packages.lock           # Lockfile: installed versions
├── 1/                          # Generation 1 (after first update)
│   ├── rootfs                  # New BTRFS snapshot
│   ├── installed_packages      # New state
│   ├── enabled_services        # New services
│   └── packages.lock           # New lockfile
├── 2/                          # Generation 2
├── current/ → ../1             # Symlink to current generation
└── .generation                 # Current active generation number
```

**Key Properties**:
- **Immutable**: Once created, generation N is never modified
- **Numbered**: Sequential 0, 1, 2, ... (monotonically increasing)
- **Point-in-time**: Each generation is complete snapshot
- **Btrfs-backed**: rootfs is actual Btrfs subvolume snapshot
- **Bootable**: Can boot into any generation via kernel parameter

### 4.2 How Kodos Manages Immutability and Rollback

**Creation of New Generation** (core.py - `create_next_generation()`):
```python
def create_next_generation(boot_part, root_part, generation):
    # 1. Create new Btrfs snapshot
    exec(f"btrfs subvolume snapshot / /kod/generations/{generation}/rootfs")
    
    # 2. Mount generation at /.next_current
    exec(f"mount -o subvol=generations/{generation}/rootfs {root_part} /.next_current")
    
    # 3. Update generation metadata
    with open(f"/.next_current/.generation", "w") as f:
        f.write(str(generation))
    
    return "/.next_current"
```

**Metadata Storage** (core.py):
```python
def store_packages_services(state_path, packages_to_install, system_services):
    # Store as JSON in generation directory (never modified after creation)
    with open(f"{state_path}/installed_packages", "w") as f:
        f.write(json.dumps(packages_to_install, indent=2))
    
    with open(f"{state_path}/enabled_services", "w") as f:
        for service in system_services:
            f.write(service + "\n")
```

**Rollback Process**:
1. Reboot and select previous generation in bootloader
2. Kernel boots into `/kod/generations/N-1/rootfs`
3. `.generation` file indicates current generation
4. System automatically loads correct packages and services
5. **No recovery needed** - complete filesystem state restored

**Immutability Guarantees**:
- Generation N directory is **never modified** after creation
- All data for generation N is **self-contained** in its directory
- Corrupting generation N does **not affect** other generations
- Deleting generation N is **safe** (can always rollback to N-1)

### 4.3 How Package Metadata is Stored in Generations

**Per-Generation Metadata Files**:

| File | Format | Content | Purpose |
|------|--------|---------|---------|
| `installed_packages` | JSON | `{"packages": ["vim", "git", ...]}` | Declarative config |
| `enabled_services` | Text | One service per line | Service state |
| `packages.lock` | Text | `vim 8.2.3455-1` | Locked versions |
| `packages.db` | Binary | Package database snapshot | Version resolution |

**Example from Kodos**:
```
/kod/generations/1/
├── rootfs                       # BTRFS snapshot
├── installed_packages           # {"packages": ["neovim", "git", "htop"]}
├── enabled_services             # networkmanager\nsshd\n
└── packages.lock                # vim 8.2.3455-1\ngit 2.35.1-1\n
```

**Metadata Lifecycle**:
1. User defines packages in Lua configuration
2. `kod rebuild -n` creates new generation
3. Metadata written to `/kod/generations/N/`
4. **Generation N directory becomes immutable**
5. On next boot, `.generation` points to N
6. System loads from `/kod/generations/N/`

**Key Difference from Chisel**:
- Kodos: Per-generation, immutable, versioned
- Chisel: Single global `/kod/registry.json`, mutable, no versioning

---

## Part 5: Compatibility Analysis - Moving Registry to /var/kod

### 5.1 Proposed Architecture Change

**Current State**:
```
/kod/registry.json                 # Single, global, mutable
└── Tracks all installed packages
```

**Proposed Kodos-Style**:
```
/var/kod/current/registry.json     # For working generation (current)
/var/kod/generations/0/registry.json
/var/kod/generations/1/registry.json
/var/kod/generations/2/registry.json
```

### 5.2 Compatibility Issues

#### Issue 1: Chisel Has No Generation Concept

**Problem**: Chisel doesn't have generational snapshots like Kodos

**Current Model**:
- Single store `/kod/store/`
- Single registry `/kod/registry.json`
- Single symlink root `/` or configurable
- No Btrfs snapshots

**Kodos Model**:
- Per-generation Btrfs snapshots
- Immutable metadata per generation
- Bootable generations
- Complete filesystem versioning

**Impact**: 
- Moving registry to `/var/kod/current/` would create single point of failure
- No benefit without generation management
- Adds path complexity without immutability

#### Issue 2: Atomicity Not Provided by Path Change Alone

**Problem**: Moving registry doesn't solve consistency issues

**Current race condition**:
```
User A: Read /kod/registry.json → Install pkg1 → Write /kod/registry.json
User B: Read /kod/registry.json (stale!) → Install pkg2 → Write /kod/registry.json (pkg1 lost!)
```

**After moving to `/var/kod/current/registry.json`**:
```
User A: Read /var/kod/current/registry.json → Install pkg1 → Write /var/kod/current/registry.json
User B: Read /var/kod/current/registry.json (still stale!) → Install pkg2 → Write /var/kod/current/registry.json (pkg1 lost!)
```

**Solution requires**: 
- File-level locking (fcntl/flock)
- OR single-generation architecture with generation file
- Moving path alone doesn't help

#### Issue 3: Multi-User Support Requires Rethinking

**Chisel vs. Kodos**:
- Chisel: Single administrative user (root)
- Kodos: Multiple users, per-user configurations

**If supporting multi-user in `/var/kod/`**:
- Need per-user registry (like per-user Lua configs)
- OR per-generation registry (like Kodos)
- Simple path move to `/var/kod/current/` doesn't address this

#### Issue 4: Persistent State Location

**Problem**: `/var/kod/` might not exist initially

**Kodos assumption**:
- Kodos is full OS, `/var/` already exists
- `/var/kod/` created during installation

**Chisel scenario**:
- Chisel might be installed on existing system
- `/kod/` is custom chisel root
- Moving to `/var/` couples to FHS compliance
- What if `/var/` is on different filesystem?

**Current code** (config.go:87):
```go
RegistryPath: filepath.Join(baseDir, "registry.json")
```

**All paths derived from `BaseDir`**:
- If BaseDir is `/kod`, registry is `/kod/registry.json`
- If BaseDir is `/opt/chisel`, registry is `/opt/chisel/registry.json`
- Design already supports multiple Chisel instances

**Moving to `/var/kod/`**:
- Breaks assumption that metadata stays with BaseDir
- Would require separate configuration for metadata
- ComplicatesMultiple-instance support

#### Issue 5: Current Design Accepts Per-Invocation Semantics

**Chisel Design**:
```
Install pkg1 → Save registry → Print "Success"
Install pkg2 → Save registry → Print "Success"
```

Each invocation is atomic (registry saved or errors out)
- No partial success notifications
- No in-process coordination needed

**Kodos Design**:
```
Start rebuild (generation N+1 created)
  Install pkg1, pkg2, pkg3
  All succeed or all fail
  Update generation N+1 atomically
  Switch boot to generation N+1
```

One invocation manages entire generation lifecycle
- All-or-nothing semantics
- Atomic generation switch
- Rollback to previous generation

**Chisel would need**:
- Concept of "generation being built"
- Metadata in `/var/kod/current/` vs `/var/kod/staging/`
- Atomic switch only on success
- Significantly more complex

---

### 5.3 Feasibility Assessment

#### INCOMPATIBLE ARCHITECTURES

**Key Mismatch**: Chisel has invocation-based atomicity, Kodos has generation-based atomicity

| Aspect | Chisel | Kodos | Compatible? |
|--------|--------|-------|-------------|
| Atomicity | Per-invocation | Per-generation | NO |
| Snapshots | None | Btrfs-based | NO |
| Bootability | N/A | Core feature | NO |
| Multi-user | Unsupported | Supported | NO |
| Metadata versioning | None | Versioned | NO |
| Rollback | Manual | Automatic | NO |

#### WHAT WOULD NEED TO CHANGE

To truly adopt Kodos-style architecture, Chisel would need:

1. **Generate Architecture**:
   - Implement concept of "generations" (currently missing)
   - Choose: Btrfs snapshots or directory-based versioning
   - Modify store to use generation-specific paths

2. **Immutability**:
   - Make generation directories immutable after creation
   - Prevent modification of past generations
   - Add validation on generation boundary

3. **Atomic Switching**:
   - Implement "staging" generation for builds
   - Only finalize on success
   - Rollback on failure

4. **Metadata Versioning**:
   - Store registry per-generation
   - Track version history
   - Implement diff/rollback

5. **Concurrency Control**:
   - Add file locking to prevent simultaneous invocations
   - OR enforce single-generation-at-a-time rule
   - OR implement per-generation locks

#### MINIMAL VIABLE CHANGE

To address **only** registry safety without full Kodos adoption:

**Option A: File-Level Locking** (Minimal, 2-3 hours work)
```go
func (r *Registry) Save() error {
    // Add fcntl write lock
    // Prevents concurrent writes
    // Minimal code change: just wrap os.WriteFile with lock
}
```

**Option B: Directory-Per-Generation** (Medium, 1-2 days work)
```
/var/kod/
├── current → ../1              # Symlink to current generation
├── 1/registry.json
├── 2/registry.json
```
- Separate registry per generation
- `current` symlink points to active
- Enables rollback without Btrfs

**Option C: Generation-Based Store** (Major, 1-2 weeks work)
```
/kod/store/generations/
├── 0/{package}/{version}/files
├── 1/{package}/{version}/files
/var/kod/
├── current → 1
├── 1/registry.json
├── 1/packages.lock
```
- Full Kodos-style architecture
- Most compatible
- Most work required

---

### 5.4 Recommendation

#### IMMEDIATE (For Current Chisel 2.0):

**Do NOT move registry.json to `/var/kod`** without additional changes:

**Reasons**:
- Path move alone provides no benefits
- Introduces complexity (must check both locations)
- Breaks current single-invocation atomicity assumptions
- Requires multi-generation support that doesn't exist
- Makes recovery harder (registry not co-located with store)

#### SHORT-TERM IMPROVEMENTS (1-2 weeks):

1. **Add File-Level Locking**:
   - Wrap registry reads/writes with fcntl locks
   - Prevent concurrent modifications
   - Enables multi-user support
   - ~50 lines of code

2. **Add Corruption Detection**:
   - Verify registry JSON before loading
   - Create backup before writing
   - Recovery instructions if corrupted

3. **Improve Error Handling**:
   - Atomic Save: write to temp file, rename
   - Validate after write: re-read and verify
   - Rollback on validation failure

#### LONG-TERM (For Future Versions):

**If adopting generation-based architecture** (weeks of work):

1. Design new generation system
2. Choose storage backend (Btrfs vs directory-based)
3. Implement generation creation/switching
4. Move registry to per-generation (then `/var/kod/generations/N/` makes sense)
5. Add rollback support
6. Add boot-time generation selection

#### FINAL VERDICT:

**Moving registry to `/var/kod/current/` is feasible but not advisable** without broader architectural changes. The benefits (if any) don't justify the migration effort or added complexity. Focus instead on:
- File-level locking for concurrency
- Better error handling and recovery
- Keeping registry.json in `/kod/` for co-location with store

---

## Appendix A: Code Locations Reference

### Registry-Related Code

| Component | Location | Lines | Purpose |
|-----------|----------|-------|---------|
| Registry struct | `pkg/registry/registry.go` | 31-35 | In-memory package map with lock |
| Registry init | `pkg/registry/registry.go` | 37-54 | Constructor, loads existing registry |
| Registry save | `pkg/registry/registry.go` | 70-80 | Marshals and writes JSON |
| Registry load | `pkg/registry/registry.go` | 56-67 | Reads and unmarshals JSON |
| Install cmd | `internal/cli/install.go` | 483-528 | Registry update during install |
| Remove cmd | `internal/cli/remove.go` | 72-176 | Registry update during removal |
| Upgrade cmd | `internal/cli/upgrade.go` | 102-170 | Multiple registry invocations |
| List cmd | `internal/cli/list.go` | 29 | Registry read |
| Cleanup cmd | `internal/cli/cleanup.go` | 118 | Registry read |
| Config defaults | `pkg/config/config.go` | 87, 229, 292 | Registry path setup |

### Kodos Generation Code

| Component | Location | Purpose |
|-----------|----------|---------|
| `get_max_generation()` | `src/kod/core.py` | Get current generation number |
| `create_next_generation()` | `src/kod/core.py` | Create new generation snapshot |
| `store_packages_services()` | `src/kod/core.py` | Write metadata to generation |
| `load_package_lock()` | `src/kod/core.py` | Read locked versions |
| Generation structure | `src/kod/kod.py` | Main rebuild logic |

### Concurrency-Related Code

| Component | Location | Lines | Purpose |
|-----------|----------|-------|---------|
| Registry mutex | `pkg/registry/registry.go` | 34 | Thread-safe in-memory access |
| Lock usage | `pkg/registry/registry.go` | 58, 71, 84, 92, 102 | Protect map access |
| Install race | `internal/cli/install.go` | 483-528 | Single write at end (safe) |
| Remove race | `internal/cli/remove.go` | 72-176 | Single write at end (safe) |
| Upgrade race | `internal/cli/upgrade.go` | 102-170 | Multiple writes (risky) |

---

## Appendix B: Recovery Procedures

### Scenario 1: Registry.json Corrupted

**Detection**:
```
$ chisel list
Error: failed to load registry: invalid character 'x' looking for beginning of value
```

**Recovery**:
1. Identify backup location (e.g., from git):
   ```
   git -C /etc/chisel show HEAD:registry.json > /tmp/registry-backup.json
   ```
2. Restore registry:
   ```
   cp /tmp/registry-backup.json /kod/registry.json
   ```
3. Verify:
   ```
   chisel list
   ```

**Prevention**:
- Add cron job to backup registry.json daily
- Commit registry.json to version control
- Add checksum validation to chisel

### Scenario 2: Registry.json Lost

**Detection**:
```
$ chisel list
No packages installed
```

But packages exist in store and work via symlinks.

**Recovery**:
Option A - Use store scan:
```
chisel scan-store  # Hypothetical command to rebuild registry
```

Option B - Manual reinstall:
```
chisel remove <all-packages>   # Clean up symlinks
chisel install <all-packages>  # Reinstall and recreate registry
```

**Limitation**: Cannot determine if packages from official repo or AUR

### Scenario 3: Partial Install Failed

**Symptom**:
```
$ chisel install pkg1 pkg2 pkg3
# Fails partway through
Error: failed to create symlink for /usr/bin/cmd: file exists

$ chisel list
pkg1 ✓
pkg2 ✗ (listed but symlinks missing)
pkg3 ✗ (not installed)
```

**Recovery**:
1. Check which symlinks exist:
   ```
   ls -la /usr/bin/pkg2*
   ```
2. Manually create missing symlinks or remove and retry:
   ```
   chisel remove pkg2
   chisel install pkg2 --force
   ```

### Scenario 4: Registry Out of Sync with Filesystem

**Symptom**:
```
$ chisel remove pkg1
Error: symlink does not exist at /usr/bin/cmd

$ chisel list
pkg1 (listed but actually gone)
```

**Cause**: Symlinks deleted externally after installation

**Recovery**:
```
chisel remove pkg1 --force  # Skip symlink checks
chisel install pkg1         # Reinstall if needed
```

---

## Conclusion

The current registry.json design is **appropriate for Chisel's architecture**. Moving it to `/var/kod/` would provide **no additional benefits** without simultaneously implementing Kodos-style generations, which would be a major architectural overhaul.

**Recommended focus**: Add file-level locking and improve error handling, rather than reorganizing file locations.

