# Registry.json - Quick Reference

## Current Usage

| Aspect | Details |
|--------|---------|
| **Location** | `/kod/registry.json` (configurable) |
| **Format** | JSON map: `{"package_name": Package{...}}` |
| **Size** | ~1KB per 5 packages (text-based) |
| **I/O Pattern** | Read once per command, write once at end |
| **Frequency** | Every install/remove/list/upgrade/cleanup |
| **Thread-Safe** | Yes (sync.RWMutex in-process) |
| **Inter-Process Safe** | NO (no file-level locking) |
| **Persistence** | Yes (survives reboots, upgrades) |

## Data Structure

```go
type Package struct {
    Name         string   // "vim"
    Version      string   // "8.2.3455-1"
    Source       string   // "official" or "aur"
    Repository   string   // "core", "extra", "community", "aur"
    Files        []string // All extracted files
    Executables  []string // Paths in usr/bin, usr/sbin
    Dependencies []string // Tracked dependencies
    InstallDate  string   // RFC3339 timestamp
    UpdateDate   string   // RFC3339 (optional)
}
```

## Command Lifecycle

### Install Command
```
1. Load registry.json
2. For each package: reg.AddPackage(pkg) [memory operation]
3. Extract packages, create symlinks, generate wrappers
4. WRITE: reg.Save() [single disk write]
```

### Remove Command
```
1. Load registry.json
2. Check dependencies
3. Remove symlinks, wrappers
4. For each package: reg.RemovePackage(name) [memory operation]
5. WRITE: reg.Save() [single disk write]
```

### List Command
```
1. Load registry.json
2. reg.ListPackages() [memory operation]
3. Display packages
```

### Upgrade Command
```
1. Load registry.json
2. For each package needing upgrade:
   - Call InstallCommand.Run() [includes Save()]
   - Load registry again [stale data possible]
```

## Concurrency Issues

### Race Condition Example

```
User A:                          User B:
1. Load registry.json (pkg1)     
2. Install pkg1                  
3. Write registry.json (pkg1)    
                                 1. Load registry.json (empty!)
                                 2. Install pkg2
                                 3. Write registry.json (only pkg2!)
Result: pkg1 is lost from registry
```

### Why It Happens
- No file-level locking (fcntl/flock)
- No TOCTOU (Time-of-Check-Time-of-Use) protection
- In-process RWMutex only protects single invocation

## What Happens If Lost/Corrupted?

### If Lost
- **Packages still work** (store intact, symlinks valid)
- Must rebuild registry by reinstalling packages
- Source info (official vs aur) cannot be recovered

### If Corrupted
- **Commands fail** with JSON parse error
- **Recovery**: Restore from backup or reinstall
- **Prevention**: Backup regularly, use version control

## Atomicity Issues

### Partial Failure Scenario
```
1. Resolve dependencies ✓
2. Download packages ✓
3. Extract packages ✓
4. Create symlinks ✗ [fails on file conflict]
5. Generate wrappers ✓
6. Update registry ✓ [PROBLEM: registry says installed but symlink missing]
```

### Why It's a Problem
- `reg.Save()` happens even if symlinks fail
- Registry state ≠ filesystem state
- Next remove command might fail

## Multi-User Support

| Scenario | Result |
|----------|--------|
| root runs `chisel install pkg1` | Works ✓ |
| root runs `chisel install pkg2` while pkg1 installing | Lost pkg1 ✗ |
| Non-root runs `chisel install` | Permission denied ✗ |
| Multiple root sessions simultaneously | Data corruption ✗ |

**Design Assumption**: Single administrative user (root) runs all commands

## Comparison: Chisel vs Kodos

| Feature | Chisel | Kodos |
|---------|--------|-------|
| Generations | None | Yes (Btrfs snapshots) |
| Registry Location | `/kod/registry.json` | `/var/kod/generations/N/` |
| Mutability | Mutable | Immutable after creation |
| Atomicity | Per-invocation | Per-generation |
| Rollback | Manual | Automatic (reboot) |
| Multi-user | Unsupported | Supported |
| Bootable | No | Yes |
| Versioning | None | Versioned per generation |
| File Locks | No | Not needed (immutable) |

## Recommended Improvements

### SHORT-TERM (1-2 weeks, ~100-200 lines code)

1. **File-Level Locking**
   ```go
   // In registry.Save()
   lock, err := AcquireExclusiveLock(r.path)
   defer lock.Unlock()
   return os.WriteFile(...)
   ```
   - Enables multi-user support
   - Prevents concurrent modifications

2. **Corruption Detection**
   ```go
   // Before Load()
   BackupRegistry(r.path)  // Create .backup file
   
   // After Save()
   VerifyRegistry(r.path)  // Re-read and validate
   ```
   - Easy recovery from backup
   - Catches write failures early

3. **Atomic Writes**
   ```go
   // Instead of WriteFile()
   WriteToTemp(data)
   ValidateTemp()
   os.Rename(temp, r.path)  // Atomic rename
   ```
   - Crash-safe writes
   - No partial files

### LONG-TERM (If needed - weeks of work)

- Design generation system (directory-based or Btrfs)
- Implement generation creation/switching
- Move registry per-generation
- Add immutability enforcement
- Add rollback support

## Code Locations

- Registry implementation: `/pkg/registry/registry.go:31-163`
- Registry usage: `/internal/cli/{install,remove,upgrade,cleanup}.go`
- Config: `/pkg/config/config.go:87,229,292`
- Tests: `/pkg/registry/registry_test.go`

## Recovery Procedures

### Lost Registry
1. Check `/kod/store/` for installed packages
2. Rebuild by reinstalling: `chisel remove all; chisel install all`
3. Or restore from git: `git checkout HEAD -- registry.json`

### Corrupted Registry
1. Restore backup: `cp /kod/registry.json.backup /kod/registry.json`
2. Or start fresh: `rm /kod/registry.json`
3. Reinstall packages if needed

### Registry Out of Sync
1. Check symlinks manually: `ls -la /usr/bin/ | grep <package>`
2. Use `--force` flag to skip checks: `chisel remove --force <pkg>`
3. Reinstall if needed: `chisel install <pkg>`

## Design Rationale

**Why single file?**
- Simple design, no versioning complexity
- All registry data in one place
- Easy to backup and restore
- Works with existing chisel architecture

**Why no inter-process locks?**
- Single-user assumption (root only)
- In-process locks sufficient for intended use
- Adding locks adds complexity
- Could be added later if needed

**Why per-invocation atomicity?**
- Each command completes or fails independently
- Partial success acceptable (e.g., some symlinks created)
- Simpler than transaction support
- User can retry failed operations

## Final Notes

- Registry is NOT the source of truth for packages
- Source of truth is filesystem (store + symlinks)
- Registry is convenience for tracking metadata
- Packages work even if registry is lost
- Main risk is inconsistency (registry ≠ filesystem)
- Most deployments: single-user, single-invocation (safe)
- Concurrent multi-user access: not supported (add locking)

