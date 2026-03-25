# Critical Architecture Decisions

This document analyzes the major architectural decisions made in the packmgr cross-distribution design, presenting alternatives considered, trade-offs, and rationale for the chosen approach.

**Version:** 4.0 (Cross-Distribution Architecture)  
**Date:** 2026-03-21

## Table of Contents

1. [Decision 1: Cross-Distribution Support Strategy](#decision-1-cross-distribution-support-strategy)
2. [Decision 2: Library Dependency Strategy](#decision-2-library-dependency-strategy)
3. [Decision 3: Binary Execution Method](#decision-3-binary-execution-method)
4. [Decision 4: Database Management Approach](#decision-4-database-management-approach)
5. [Decision 5: ALPM Usage Strategy](#decision-5-alpm-usage-strategy)
6. [Decision 6: Programming Language (Go vs Python)](#decision-6-programming-language-go-vs-python)
7. [Decision 7: Remove Generation Management](#decision-7-remove-generation-management)
8. [Decision 8: Package Storage Model](#decision-8-package-storage-model)
9. [Decision 9: Package Manager Integration](#decision-9-package-manager-integration)
10. [Decision 10: Symlink Strategy](#decision-10-symlink-strategy)
11. [Decision 11: Registry Format](#decision-11-registry-format)
12. [Decision 12: Store Cleanup Strategy](#decision-12-store-cleanup-strategy)
13. [Decision 13: Package Script Execution](#decision-13-package-script-execution)

---

## Decision 1: Cross-Distribution Support Strategy

### Context
**NEW in v4.0:** Need to decide whether packmgr should work ONLY on Arch Linux or support multiple distributions (Ubuntu, Fedora, Debian, etc.).

### Options Considered

#### Option A: Arch Linux Only

**Implementation**: Traditional approach - use system's `/var/lib/pacman`, integrate with host pacman.

**Pros**:
- Simpler implementation
- Can use host's glibc and system libraries
- Smaller storage overhead (no duplicate libraries)
- Native integration with pacman

**Cons**:
- **Limited audience**: Only Arch users benefit
- **Can't help Ubuntu/Fedora users** wanting bleeding-edge tools
- Competes directly with pacman (why would users switch?)
- Missing opportunity to solve real problem (stable distro + latest packages)

#### Option B: Cross-Distribution Support (CHOSEN)

**Implementation**: Bring Arch packages to ANY Linux distribution.

**Pros**:
- **Much larger audience**: Ubuntu, Fedora, Debian, Mint users (~70% of Linux desktop market)
- **Solves real problem**: Stable distro users want bleeding-edge tools without full system upgrade
- **Unique value**: Not competing with host package managers, supplementing them
- **Developer appeal**: Latest versions of dev tools on LTS distributions
- **No system contamination**: Isolated from host package manager

**Cons**:
- More complex implementation (wrapper scripts, library isolation)
- 2-3x storage overhead (ship ALL dependencies including glibc)
- Requires careful library path management
- Need multi-distribution testing (Docker/Podman)

### Decision Rationale

**CHOSEN: Cross-Distribution Support**

Critical insight: Packmgr's unique value is bringing Arch packages to non-Arch systems.

**Why this is the right choice:**
1. **Market size**: 10x larger audience (Ubuntu/Fedora/Debian vs Arch)
2. **Real need**: Ubuntu 22.04 LTS users stuck on Python 3.10, want 3.12 from Arch
3. **No competition**: Not replacing apt/dnf, supplementing them
4. **Differentiation**: Only tool that brings Arch packages everywhere
5. **Storage is cheap**: 200MB vs 60MB acceptable trade-off for "works everywhere"

**Use cases unlocked:**
- Ubuntu LTS + latest development tools
- Fedora + specific Arch package versions
- Debian Stable + bleeding-edge vim/emacs
- CI/CD with consistent tooling across distributions

---

## Decision 2: Library Dependency Strategy

### Context
**NEW in v4.0:** Arch binaries compiled against glibc 2.39 will crash on Ubuntu 22.04 (glibc 2.35). How do we handle library dependencies?

### Options Considered

#### Option A: Use Host Libraries

**Implementation**: Install Arch binaries, let them use Ubuntu's/Fedora's system libraries.

**Pros**:
- Minimal storage overhead
- No library duplication
- Simpler installation

**Cons**:
- **FATAL FLAW**: Incompatible library versions cause crashes
- **Example**: Arch vim + Ubuntu glibc 2.35 = immediate segfault
- **Symbol errors**: Missing GLIBC_2.38, GLIBC_2.39 symbols
- **Unpredictable**: Different distros, different failures
- **Debugging nightmare**: Hard to diagnose library version mismatches

**Why Rejected**: Doesn't work. Testing confirmed immediate crashes.

#### Option B: Full Dependency Isolation (CHOSEN)

**Implementation**: Install ALL dependencies from Arch (glibc, gcc-libs, ncurses, etc.)

**Pros**:
- **Guaranteed compatibility**: Arch binaries + Arch libraries = always works
- **Predictable behavior**: Same libraries on all distributions
- **No host contamination**: Completely isolated from host package manager
- **Easier debugging**: Known library versions
- **Future-proof**: New distros automatically supported

**Cons**:
- **2-3x storage overhead**: vim = 60MB → 200MB with all deps
- **Download size**: Larger initial downloads
- **Complexity**: Must ship and manage system libraries

**Storage Example**:
```
vim package alone:  60 MB
With ALL dependencies from Arch:
  + glibc (22 MB)
  + gcc-libs (31 MB)
  + ncurses (1.2 MB)
  + other deps (85 MB)
  = Total: 201 MB
```

#### Option C: Container-Based (Docker/Flatpak)

**Implementation**: Run Arch packages in containers.

**Pros**:
- Complete isolation
- Proven technology

**Cons**:
- **Heavy runtime**: Docker daemon, containerd
- **Slow startup**: Container initialization overhead
- **Complex integration**: Accessing host files
- **User experience**: Not seamless
- **Resource usage**: More memory per app

**Why Rejected**: Too heavy. We want seamless `vim file.txt`, not `docker run arch-vim vim file.txt`.

#### Option D: Static Binaries

**Implementation**: Use statically-linked binaries.

**Pros**:
- No library dependencies
- Simple deployment

**Cons**:
- **Not available**: Most Arch packages don't offer static builds
- **Huge size**: Static binaries 10x larger
- **Plugin issues**: Dynamic loading doesn't work
- **Limited packages**: Only works for simple CLI tools

**Why Rejected**: Not feasible for most packages.

### Decision Rationale

**CHOSEN: Full Dependency Isolation**

Accept 2-3x storage overhead for "works everywhere" guarantee.

**Key insight**: Storage is cheap (~$20/TB), compatibility is priceless.

**Why this works:**
1. **One-time cost**: Download large once, works forever
2. **Predictable**: Same behavior on Ubuntu, Fedora, Debian
3. **Maintainable**: No per-distro testing or fixes
4. **User-friendly**: Just works, no configuration needed

**Trade-off accepted:**
- 200 MB vs 60 MB for vim
- BUT: vim works identically on ALL distributions
- Storage overhead worth the universality

---

## Decision 3: Binary Execution Method

### Context
**NEW in v4.0:** Arch binaries need to find Arch libraries, not host libraries. How do we ensure correct library loading?

### Options Considered

#### Option A: Direct Symlinks (FAILS)

**Implementation**: `/usr/bin/vim` → `/kod/store/vim/9.0/usr/bin/vim`

**Pros**:
- Simple
- No overhead

**Cons**:
- **FATAL**: Binary loads host libraries via default paths (/lib, /usr/lib)
- **Crashes**: Arch binary + Ubuntu libraries = segfault
- **LD_LIBRARY_PATH not set**: Linker doesn't know about `/kod/store/`

**Why Rejected**: Doesn't work. Binary loads wrong libraries.

#### Option B: Wrapper Scripts (CHOSEN)

**Implementation**: Two-tier system:
1. Symlink: `/usr/bin/vim` → `/kod/wrappers/vim`
2. Wrapper sets `LD_LIBRARY_PATH` then execs actual binary

**Pros**:
- **Works reliably**: Forces correct library paths
- **Simple to understand**: Just a shell script
- **Easy to debug**: `cat /kod/wrappers/vim` shows exact setup
- **Flexible**: Can add environment variables, error checking
- **Per-package control**: Each wrapper customized for its dependencies

**Cons**:
- Slight startup overhead (~2-5ms for shell script execution)
- Extra files to manage (`/kod/wrappers/`)
- Indirect execution (symlink → wrapper → binary)

**Wrapper Example**:
```bash
#!/bin/bash
export LD_LIBRARY_PATH="/kod/store/vim/9.0/usr/lib:/kod/store/glibc/2.39/usr/lib:$LD_LIBRARY_PATH"
exec /kod/store/vim/9.0/usr/bin/vim "$@"
```

#### Option C: PAT CHELF/patchelf

**Implementation**: Modify ELF binaries to hardcode `/kod/store/` paths.

**Pros**:
- No wrapper overhead
- Direct execution

**Cons**:
- **Modifies binaries**: Breaks signatures, checksums
- **Fragile**: Path length limits in ELF format
- **Complex**: Need to patch every binary and .so file
- **Maintenance nightmare**: Re-patch on every package update
- **Breaking changes**: Could break binaries

**Why Rejected**: Too invasive, too fragile.

#### Option D: Custom Dynamic Linker

**Implementation**: Use custom `ld.so` configuration.

**Pros**:
- System-level solution

**Cons**:
- **Affects host system**: Risky
- **Complex**: `/etc/ld.so.conf.d/` manipulation
- **Conflicts**: Could break host packages
- **Requires root**: Even more privileges

**Why Rejected**: Too dangerous to host system.

### Decision Rationale

**CHOSEN: Wrapper Scripts**

Best balance of simplicity, reliability, and debuggability.

**Why wrappers win:**
1. **Dead simple**: 5-line shell script anyone can understand
2. **Reliable**: LD_LIBRARY_PATH always works
3. **Debuggable**: `cat wrapper.sh` shows exactly what happens
4. **Safe**: No binary modification, no system-wide changes
5. **Flexible**: Easy to add debugging, error checks later

**Overhead analysis:**
- Wrapper execution: ~2ms (fork + exec bash)
- User won't notice on vim startup (~50ms total)
- Worth it for guaranteed correctness

---

## Decision 4: Database Management Approach

### Context
**NEW in v4.0:** Need Arch package databases (core.db, extra.db) to query packages. Should we integrate with host pacman or maintain separate databases?

### Options Considered

#### Option A: Use Host Pacman Databases

**Implementation**: Read from `/var/lib/pacman/sync/`

**Pros**:
- No separate sync needed
- Automatic updates (when user runs `pacman -Sy`)

**Cons**:
- **Doesn't exist on Ubuntu/Fedora**: Fatal for cross-distribution
- **Tied to host**: Can't work independently
- **Version conflicts**: Host might have different Arch mirror versions

**Why Rejected**: Doesn't work on non-Arch systems (our primary use case).

#### Option B: Separate Database Sync (CHOSEN)

**Implementation**: Download databases to `/kod/db/` via `packmgr sync`

**Pros**:
- **Works everywhere**: Ubuntu, Fedora, Debian don't need pacman installed
- **Independent**: Packmgr doesn't depend on host package manager
- **User control**: Explicit sync, not automatic
- **Custom mirrors**: Can use fastest/preferred mirror
- **Isolation**: No conflicts with host package manager

**Cons**:
- Extra command (`packmgr sync`) before first use
- Databases can become stale if user forgets to sync
- Additional storage (~5-10 MB for databases)

**Implementation**:
```bash
# User explicitly syncs databases
packmgr sync

# Downloads from Arch mirror:
#   https://mirror.rackspace.com/archlinux/core/os/x86_64/core.db
#   https://mirror.rackspace.com/archlinux/extra/os/x86_64/extra.db
# 
# Saves to:
#   /kod/db/core.db
#   /kod/db/extra.db
```

#### Option C: Embedded Database

**Implementation**: Bundle package database in packmgr binary.

**Pros**:
- No sync needed

**Cons**:
- **Stale immediately**: Binary compiled yesterday, packages released today
- **Huge binary**: 5-10 MB database in every binary
- **Requires recompilation**: For every database update
- **Impractical**: Defeats purpose of package manager

**Why Rejected**: Databases must be fresh, not compiled-in.

### Decision Rationale

**CHOSEN: Separate Database Sync**

Only option that works for cross-distribution use case.

**Why explicit sync:**
1. **Supplementary tool**: Packmgr supplements host PM, not replaces it
2. **Infrequent use**: Users don't install Arch packages daily
3. **User control**: User decides when to check for updates
4. **Network efficiency**: No automatic syncs eating bandwidth

**User workflow:**
```bash
# Once a week or before installing packages:
packmgr sync

# Then install as needed:
packmgr install vim
packmgr install python
```

---

## Decision 5: ALPM Usage Strategy

### Context
**NEW in v4.0:** Should we use libalpm (Arch's library) or implement dependency resolution ourselves?

### Options Considered

#### Option A: DIY Dependency Resolution

**Implementation**: Parse `.PKGINFO`, manually resolve dependencies, implement conflict detection.

**Pros**:
- No libalpm dependency
- Full control over logic

**Cons**:
- **4-6 weeks of work**: Complex algorithm
- **2,000-3,000 lines of code**: Hard to maintain
- **High bug risk**: Edge cases (optdepends, conflicts, provides, replaces)
- **Reinventing wheel**: ALPM already does this perfectly
- **Testing nightmare**: Need extensive test suite
- **Feature gaps**: ALPM has 15 years of bug fixes

**Complexity examples:**
- `provides` virtual packages
- `conflicts` detection
- Circular dependencies
- Version constraints (`>=`, `<`, `=`)
- Optional dependencies
- Package replacement logic

**Why Rejected**: Not worth 4-6 weeks + maintenance burden.

#### Option B: Use go-alpm Library (CHOSEN)

**Implementation**: Use `github.com/Jguer/go-alpm/v2` with custom root `/kod/`.

**Pros**:
- **Battle-tested**: Used by yay (most popular AUR helper)
- **1-2 weeks implementation**: Just wrapper code
- **500-800 lines vs 2000-3000**: Much simpler
- **All features**: Dependency resolution, conflicts, provides, etc.
- **Bug-free**: 15 years of pacman development
- **Easy to understand**: Simple API calls

**Cons**:
- Requires libalpm installed (`apt install libalpm-dev` on Ubuntu)
- CGO dependency (but acceptable)
- Tied to libalpm API (but stable)

**Implementation**:
```go
// Initialize ALPM with /kod root instead of /
handle, err := alpm.Initialize("/kod", "/kod/db")

// Use normally for queries, dependency resolution
pkg, err := handle.SyncDbByName("core").Pkg("vim")
deps, err := pkg.ComputeRequiredBy()
```

**Key insight**: ALPM can work with ANY root directory, not just `/`. Perfect for our isolated `/kod/` approach.

#### Option C: Call pacman Binary

**Implementation**: Shell out to `pacman` command.

**Pros**:
- No library dependency

**Cons**:
- **Doesn't exist on Ubuntu/Fedora**: Fatal
- **Awkward API**: Parsing text output
- **Fragile**: Output format changes break us
- **Slow**: Process spawning overhead

**Why Rejected**: Doesn't work on non-Arch systems.

### Decision Rationale

**CHOSEN: Use go-alpm Library**

Saves 4-6 weeks, reduces code by 70%, eliminates entire class of bugs.

**Time/effort comparison:**
```
DIY Dependency Resolution:
- Time: 4-6 weeks
- Code: 2,000-3,000 lines
- Bugs: High risk (complex algorithm)
- Maintenance: Ongoing (new edge cases)

Using go-alpm:
- Time: 1-2 weeks
- Code: 500-800 lines
- Bugs: Low risk (battle-tested)
- Maintenance: Minimal (library updated by maintainers)
```

**Decision is obvious**: Use the library. Not even close.

**Bonus**: libalpm installable on all distributions:
- Ubuntu/Debian: `apt install libalpm-dev`
- Fedora: `dnf install libalpm-devel`
- Arch: Already installed

---

## Decision 6: Programming Language (Go vs Python)

### Context
Need to choose implementation language for the package manager.

### Options Considered

#### Option A: Go (CHOSEN)

**Implementation**: Pure Go with go-alpm bindings.

**Pros**:
- **Single binary**: No runtime dependencies (except libalpm.so)
- **Fast**: Compiled, native performance
- **Excellent concurrency**: Goroutines for parallel downloads
- **Static typing**: Catch errors at compile time
- **Cross-compilation**: Easy to build for different architectures
- **No chicken-and-egg problem**: Can bootstrap without Python
- **Mature ALPM bindings**: `github.com/Jguer/go-alpm/v2` (31+ importers)
- **Better error handling**: Explicit error returns

**Cons**:
- Slightly more verbose than Python
- Longer compile times (though still fast)
- CGO dependency for ALPM

**Performance Comparison**:
```
Operation          Go        Python
─────────────────────────────────────
Binary size        5-10 MB   N/A (runtime)
Startup time       <10ms     ~100ms
Memory (idle)      5-10 MB   30-50 MB
Concurrency        Goroutines asyncio (complex)
Type safety        Compile    Runtime (mypy)
```

#### Option B: Python

**Pros**:
- Rapid development
- Rich ecosystem
- Easy prototyping

**Cons**:
- **Python runtime required**: Chicken-and-egg problem
- **Slower**: Interpreted language
- **GIL limitations**: True parallelism limited
- **Runtime errors**: Type errors caught late
- **Deployment**: Need Python + dependencies

**Why Rejected**: Python itself is a package that could break. If Python breaks, the package manager breaks → system unrecoverable.

#### Option C: Rust

**Pros**:
- Memory safety
- Excellent performance
- Modern language

**Cons**:
- Steeper learning curve
- Longer development time
- No mature ALPM bindings
- Overkill for this use case

**Why Rejected**: Development time too long, no good ALPM bindings.

### Decision Rationale

**CHOSEN: Go**

Best choice because:
1. **No runtime dependency**: Single static binary (except libalpm.so which pacman needs anyway)
2. **Performance**: Fast enough for package operations
3. **Mature ecosystem**: go-alpm/v2 is production-ready (used by yay)
4. **Development speed**: Faster than C/Rust, safer than Python
5. **Deployment**: Just copy one binary
6. **Maintainability**: Static typing prevents many bugs

### Implementation Impact

```go
// Example: Simple package installation
package main

import (
    "github.com/Jguer/go-alpm/v2"
    "log"
)

func main() {
    h, _ := alpm.Initialize("/", "/var/lib/pacman")
    defer h.Release()
    
    db, _ := h.RegisterSyncDB("core", alpm.SigLevel(0))
    pkg := db.Pkg("bash")
    
    log.Printf("Installing %s %s", pkg.Name(), pkg.Version())
}
```

---

## Decision 2: Remove Generation Management

### Context
Original design included generation management (snapshots, rollback). Should we keep this complexity?

### Options Considered

#### Option A: Remove Generations (CHOSEN)

**Implementation**: Simple package store with symlinks. No generation snapshots.

**Pros**:
- **Dramatically simpler**: 50% less code
- **No btrfs requirement**: Works on any filesystem
- **Faster development**: 6-8 weeks vs 20+ weeks
- **Easier to understand**: Clear mental model
- **Lower risk**: Less can go wrong
- **Realistic MVP**: Actually deliverable
- **Storage efficient**: Only one copy of each package version

**Cons**:
- No system snapshots
- No instant rollback to previous state
- Cannot boot into different configurations
- No NixOS-like generation switching

**Scope Reduction**:
```
Original:  15,000+ lines of code estimated
Simplified: 3,000-5,000 lines of code
```

#### Option B: Keep Generations

**Pros**:
- Full rollback capability
- System snapshots
- NixOS-like features

**Cons**:
- **Too complex for v1**: 20+ week timeline
- **Btrfs requirement**: Limited filesystem choice
- **Boot integration needed**: High risk
- **Many edge cases**: Hard to get right
- **May never finish**: Scope too large

**Why Rejected**: Too ambitious for initial version. Can add in v2.0 if needed.

### Decision Rationale

**CHOSEN: Remove Generations**

Optimal because:
1. **Ship faster**: 6-8 weeks vs 20+ weeks
2. **Lower risk**: Simpler system = fewer bugs
3. **Core value intact**: Package management with deduplication works
4. **Extensible**: Can add generations in v2.0 if users want it
5. **Realistic**: Actually achievable

Quote: "Perfect is the enemy of good" - ship something that works.

---

## Decision 3: Package Storage Model

### Context
How to store packages efficiently across the system.

### Options Considered

#### Option A: Centralized Store with Symlinks (CHOSEN)

**Implementation**: 
- Extract all packages to `/kod/store/<pkg>/<version>/`
- System files are symlinks to store

**Pros**:
- **Maximum visibility**: Can see all package files
- **Easy inspection**: Browse `/kod/store/` to see everything
- **Deduplication**: Each version stored once
- **Simple rollback**: Just re-point symlinks (future feature)
- **Clear ownership**: Symlink = managed by packmgr
- **Space efficient**: Shared files across "virtual" generations (future)

**Cons**:
- Many symlinks to manage
- Some tools might not follow symlinks properly
- Initial setup more complex

**Example**:
```
Store:
/kod/store/bash/5.2.26-1/usr/bin/bash [actual file, 1.2MB]

Symlinks:
/usr/bin/bash -> /kod/store/bash/5.2.26-1/usr/bin/bash

Disk usage: 1.2MB + 4KB (symlink) = ~1.2MB
```

#### Option B: Traditional In-Place Installation

**Implementation**: Extract directly to /usr/, /etc/, etc.

**Pros**:
- Standard approach
- No symlinks
- Maximum compatibility

**Cons**:
- No central location to inspect packages
- Cannot easily switch versions
- Cleanup is complex
- No deduplication

**Why Rejected**: Doesn't provide unique value over pacman.

### Decision Rationale

**CHOSEN: Centralized Store**

Best approach because:
1. **Inspectable**: Easy to see what's installed and where
2. **Organized**: All packages in one place
3. **Efficient**: Deduplicated storage
4. **Extensible**: Foundation for future generation support
5. **Unique value**: Different from pacman

---

## Decision 4: Package Manager Integration

### Context
How to integrate with Arch Linux package infrastructure.

### Options Considered

#### Option A: go-alpm Bindings (CHOSEN)

**Implementation**: Use `github.com/Jguer/go-alpm/v2`

**Pros**:
- **Native Arch support**: Direct ALPM integration
- **Mature library**: Used by yay (popular AUR helper)
- **Complete functionality**: All ALPM features available
- **Well maintained**: Active development
- **31+ importers**: Proven in production
- **Transaction support**: Atomic operations
- **Dependency resolution**: Built-in

**Cons**:
- CGO dependency (requires libalpm.so)
- Arch-specific (but that's the target)

**Example**:
```go
import "github.com/Jguer/go-alpm/v2"

h, _ := alpm.Initialize("/", "/var/lib/pacman")
db, _ := h.RegisterSyncDB("core", 0)
pkg := db.Pkg("bash")
deps := pkg.Depends()  // Automatic dependency resolution
```

#### Option B: Shell out to pacman

**Implementation**: Execute `pacman` commands via shell.

**Pros**:
- No library dependencies
- Always compatible

**Cons**:
- **Slow**: Subprocess overhead
- **Hard to parse**: Scraping text output
- **Fragile**: Output format changes break us
- **Limited control**: Cannot customize behavior

**Why Rejected**: Too slow and fragile.

### Decision Rationale

**CHOSEN: go-alpm**

Optimal because:
1. **Performance**: Native library calls
2. **Reliability**: Proven in production (yay uses it)
3. **Features**: Full ALPM functionality
4. **Maintenance**: Active development
5. **Integration**: Direct access to Arch package ecosystem

---

## Decision 5: Symlink Strategy

### Context
How to link system files to the package store.

### Options Considered

#### Option A: Individual File Symlinks (CHOSEN)

**Implementation**: Create symlink for each file.

```
/usr/bin/bash -> /kod/store/bash/5.2.26-1/usr/bin/bash
/usr/bin/vim  -> /kod/store/vim/9.0.1-1/usr/bin/vim
```

**Pros**:
- **Granular control**: Manage each file independently
- **Clear ownership**: Easy to see what's managed
- **Selective installation**: Can omit certain files if needed
- **Standard behavior**: Files work like normal files
- **Easy verification**: Check each symlink individually

**Cons**:
- More symlinks to create/manage
- Slightly slower installation (more syscalls)

#### Option B: Directory Bind Mounts

**Implementation**: Mount entire directories.

```bash
mount --bind /kod/store/bash/5.2.26-1/usr /usr
```

**Pros**:
- Fewer mount operations
- Atomic mounting

**Cons**:
- **Complex boot integration**: Must mount at boot
- **Fragile**: Mount failures break system
- **Hard to debug**: Less visible what's mounted
- **Persistence issues**: Mounts don't survive reboot without fstab

**Why Rejected**: Too complex and fragile.

### Decision Rationale

**CHOSEN: Individual File Symlinks**

Best approach because:
1. **Simplicity**: Just filesystem operations
2. **Debuggability**: `ls -la /usr/bin/bash` shows where it points
3. **Persistence**: Symlinks survive reboots automatically
4. **No special boot setup**: Works immediately
5. **Standard Unix**: Well-understood mechanism

---

## Decision 6: Registry Format

### Context
How to track installed packages.

### Options Considered

#### Option A: JSON Registry (CHOSEN)

**Implementation**: Single `registry.json` file.

```json
{
  "version": "1.0",
  "packages": [
    {
      "name": "bash",
      "version": "5.2.26-1",
      "files": ["/usr/bin/bash", ...]
    }
  ]
}
```

**Pros**:
- **Simple**: Easy to read and write
- **Human readable**: Can inspect with `cat`
- **Standard**: Every language has JSON support
- **Atomic writes**: Write temp + rename
- **Easy backup**: Just one file

**Cons**:
- Must load entire file
- No indexing (fine for <10k packages)

#### Option B: SQLite Database

**Implementation**: SQLite `.db` file.

**Pros**:
- Indexing and queries
- Transactions
- Scalable

**Cons**:
- **Overkill**: Don't need SQL for simple registry
- **Corruption risk**: Database corruption more complex
- **Less inspectable**: Can't just `cat` it
- **Extra dependency**: Need SQLite library

**Why Rejected**: JSON is sufficient for this use case.

### Decision Rationale

**CHOSEN: JSON**

Optimal because:
1. **Simplicity**: Easy to implement
2. **Inspectable**: Can read with any tool
3. **Sufficient**: Fast enough for expected package counts
4. **Portable**: Easy to backup/restore
5. **Atomic updates**: Write to temp file + rename

---

## Decision 7: Store Cleanup Strategy

### Context
When to remove packages from `/kod/store/`.

### Options Considered

#### Option A: Deferred Cleanup (CHOSEN)

**Implementation**: Keep packages in store after removal, cleanup manually.

```bash
packmgr remove vim        # Removes symlinks, keeps in store
packmgr cleanup --keep 3  # Manual cleanup
```

**Pros**:
- **Fast reinstall**: No re-download needed
- **Easy version switching**: Just re-point symlinks (future)
- **Safer**: Can recover if removal was mistake
- **Bandwidth saving**: Re-install from local store

**Cons**:
- Uses more disk space
- Need periodic cleanup

**Configuration**:
```yaml
auto_cleanup: false       # Default: keep in store
keep_versions: 3          # For manual cleanup
```

#### Option B: Immediate Cleanup

**Implementation**: Delete from store immediately on removal.

```bash
packmgr remove vim  # Removes symlinks AND deletes store files
```

**Pros**:
- No wasted space
- Simple

**Cons**:
- Must re-download if reinstalling
- Cannot rollback removal
- Wastes bandwidth

**Why Rejected**: Bandwidth is more precious than disk space.

### Decision Rationale

**CHOSEN: Deferred Cleanup**

Best approach because:
1. **Bandwidth savings**: Don't re-download frequently used packages
2. **Flexibility**: User controls when to clean
3. **Recovery**: Can undo removal easily
4. **Future-proof**: Enables version switching later
5. **User choice**: Can cleanup with flag if desired

---

## Decision 8: Package Script Execution

### Context
Arch packages have install scripts (pre/post install/remove). Should we execute them?

### Options Considered

#### Option A: Skip Scripts in v1 (CHOSEN)

**Implementation**: Don't execute package scripts initially.

**Pros**:
- **Simpler**: No script sandbox needed
- **Safer**: Scripts can't break system
- **Faster delivery**: Ship v1 sooner
- **Lower risk**: Fewer attack vectors
- **Most packages work**: Scripts often optional

**Cons**:
- Some packages may not fully function
- System hooks not triggered (font cache, etc.)
- Missing functionality vs pacman

**Documented Limitations**:
```
Note: packmgr v1.0 does not execute package scripts.
Packages requiring scripts may not function fully.
Use pacman for system-critical packages requiring scripts.
```

#### Option B: Execute Scripts with Sandbox

**Implementation**: Run scripts in restricted environment.

**Pros**:
- Full package functionality
- Compatible with all packages

**Cons**:
- **Complex**: Need proper sandboxing
- **Security risk**: Scripts run as root
- **Slower delivery**: Takes time to implement safely
- **Many edge cases**: Script execution is tricky

**Deferred to v1.1+**: Will implement after core functionality is solid.

### Decision Rationale

**CHOSEN: Skip Scripts in v1**

Best approach because:
1. **Ship faster**: Focus on core functionality
2. **Reduce risk**: Fewer security concerns
3. **Iterate**: Get feedback before adding scripts
4. **Most packages work**: Many don't need scripts
5. **Clear roadmap**: v1.1 = add script support

**Migration Path**:
- v1.0: No scripts (document limitations)
- v1.1: Add script execution with sandboxing
- v2.0: Full hook system

---

## Summary Table

| Decision | Chosen Option | Key Reason |
|----------|--------------|------------|
| Language | Go | Single binary, no runtime dependency |
| Generations | Remove (v1) | Ship faster, lower complexity |
| Storage Model | Centralized store + symlinks | Inspectable, organized, efficient |
| Package Manager | go-alpm/v2 | Mature, proven, full ALPM features |
| Symlink Strategy | Individual files | Simple, debuggable, persistent |
| Registry Format | JSON | Simple, inspectable, sufficient |
| Cleanup Strategy | Deferred | Bandwidth savings, flexibility |
| Script Execution | Skip in v1 | Ship faster, add later |

---

## Decision Impact Matrix

### High Impact (Core Architecture)
1. **Language (Go)**: Enables entire system
2. **Remove Generations**: Makes project achievable
3. **Storage Model**: Determines all file operations
4. **go-alpm**: Provides Arch integration

### Medium Impact (Implementation)
5. **Symlink Strategy**: Affects file management
6. **Cleanup Strategy**: Affects user experience
7. **Registry Format**: Affects performance at scale

### Lower Impact (Can Change Later)
8. **Script Execution**: Can add incrementally

---

## Comparison: v2 (Complex) vs v3 (Simplified)

| Aspect | v2.0 (Original) | v3.0 (Simplified) |
|--------|----------------|-------------------|
| Language | Python | Go |
| Timeline | 20+ weeks | 6-8 weeks |
| Code Lines | 15,000+ | 3,000-5,000 |
| Generations | Yes (btrfs) | No (future v2) |
| Boot Integration | Yes | No (future v2) |
| Dependencies | Python + many libs | Go + libalpm |
| Deployment | Complex | Single binary |
| Learning Curve | High | Low |
| Risk | High | Low |
| Deliverable | Questionable | Yes |

---

## Future Reconsiderations

Decisions that might be revisited:

1. **Generation Management (v2.0)**: Add if users want it
2. **Script Execution (v1.1)**: High priority for next version
3. **SQLite Registry (v2.0+)**: If registry grows >10MB
4. **Web UI (v2.0)**: If remote management needed
5. **AUR Support (v2.5)**: Natural extension

---

## Lessons from v2 Redesign

What we learned:

1. **Simplicity wins**: Remove 50% of features → ship 3x faster
2. **MVP first**: Get core working before adding bells and whistles
3. **Language matters**: Go's single binary >> Python's runtime
4. **Proven libraries**: Use go-alpm, don't reinvent
5. **Defer complexity**: Generations → v2, Scripts → v1.1
6. **Ship something**: Better to have basic version than nothing

The simplified design keeps the core value (centralized package store with deduplication) while removing the complexity that made v2 unshippable.

---

## Document Control

**Revision History:**

| Version | Date | Changes |
|---------|------|---------|
| 3.0 | 2026-03-21 | Simplified decisions for Go implementation |
| 2.0 | 2026-03-21 | Complex generation-based decisions |
| 1.0 | 2026-01-01 | Initial decisions |

---

*End of Critical Decisions Document*
