# Generation-Based Registry Management Specification

**Status**: Future Work - Planning Phase  
**Created**: April 6, 2026  
**Scope**: Integration of chisel package management with Kodos generation system

---

## Table of Contents

1. [Overview](#overview)
2. [Current State](#current-state)
3. [Proposed Architecture](#proposed-architecture)
4. [Implementation Strategy](#implementation-strategy)
5. [Design Decisions](#design-decisions)
6. [Open Questions](#open-questions)
7. [Future Phases](#future-phases)

---

## Overview

This specification outlines a future architecture for integrating chisel's package registry with Kodos' generation-based system. The goal is to leverage Kodos' immutable generations to eliminate registry locking issues and provide per-user package state management with full rollback capabilities.

### Key Principles

- **Immutable Generations**: Each package operation creates a new generation snapshot
- **Per-user Registries**: Each user maintains their own package registry in their home directory
- **No Registry Locking**: Immutability eliminates race conditions
- **Atomic State**: Generation captures complete package state at a point in time
- **Rollback Enabled**: Users can revert to any previous package state

---

## Current State

### Registry.json Today

**Location**: `/kod/registry.json` (or user-specific `~/.local/share/chisel/registry.json`)

**Structure**:
```json
{
  "package-name": {
    "name": "package-name",
    "version": "1.0.0-1",
    "source": "official|aur",
    "repository": "core|extra|aur",
    "files": ["usr/bin/cmd", "usr/lib/lib.so"],
    "executables": ["usr/bin/cmd"],
    "dependencies": ["dep1", "dep2"],
    "install_date": "2024-01-15T10:30:00Z",
    "update_date": "2024-03-21T14:20:00Z"
  }
}
```

**Current Issues**:
- No file-level locking → potential race conditions if multiple chisel commands run concurrently
- Not integrated with system generation/rollback mechanisms
- Requires persistence management across system upgrades
- Limited audit trail for package history

**Current Strengths**:
- Simple JSON format, human-readable
- Tracks all necessary package metadata
- Per-user support already possible via config

---

## Proposed Architecture

### 1. Registry Storage Model

#### Per-User Registry Location
```
$HOME/.local/share/chisel/registry.json
```

**Why this location:**
- Follows XDG Base Directory Specification
- Already implemented in current chisel
- Clean per-user separation
- No permission issues
- Survives generation rollbacks (lives outside generation)

#### Generation Registry Snapshot
```
/var/kod/generations/GEN_ID/metadata/registry.json
/var/kod/generations/GEN_ID/metadata/registry.snapshot.json.gz  (optional compressed)
```

**Purpose:**
- Audit trail: what packages were installed in each generation
- Rollback reference: can restore from generation if user registry corrupted
- History tracking: enables package change analysis between generations

### 2. Generation Integration Points

#### On Package Install/Remove/Upgrade

```
User Command:
  chisel-user install vim

Execution Flow:
  1. Load $HOME/.local/share/chisel/registry.json
  2. Resolve dependencies
  3. Download/build packages
  4. Extract to store
  5. Create symlinks
  6. Update registry.json with new package entry
  7. TRIGGER: kodos create-generation
       └─ Creates new generation snapshot
       └─ Stores registry.json copy in generation metadata
       └─ Generation becomes immutable
  8. Return success to user
```

#### On System Rollback

```
User Command:
  kodos switch-generation GEN_ID

Execution Flow:
  1. Kodos switches to generation GEN_ID (immutable)
  2. chisel commands detect generation change
  3. OPTION A: Auto-restore registry from generation snapshot
  4. OPTION B: User manually resync registry
  5. Package state matches generation
  6. chisel commands work with restored registry
```

### 3. Immutability Model

**Generation Root**: Read-only after creation
- Contains extracted packages
- Contains generated symlinks
- Contains wrapper scripts

**User Home**: Always mutable
- `~/.local/share/chisel/registry.json` (current/active)
- `~/.local/share/chisel/cache/` (mutable)
- `~/.config/chisel/` (mutable config)

**Rationale**: Registry needs to be accessible even when generation is read-only, allows users to reinstall packages or audit history.

### 4. Multi-User Scenario

#### Per-User Generation Chain

```
System User (root):
  Generation 1: base-system
  Generation 2: base-system + system packages
    └─ /var/kod/generations/GEN_2/metadata/registry.json (system packages)

Regular User (alice):
  User Registry: ~/.local/share/chisel/registry.json (alice's packages only)
  User Generation Chain: ~/.kodos/generations/
    └─ Each generation tied to alice's user context
    └─ Captures alice's installed packages

Regular User (bob):
  User Registry: ~/.local/share/chisel/registry.json (bob's packages only)
  User Generation Chain: ~/.kodos/generations/
    └─ Separate from alice's generations
```

---

## Implementation Strategy

### Phase 1: Planning & Preparation (Current)

- [x] Clarify generation creation triggers
- [x] Confirm per-user registry location
- [x] Understand immutability model
- [x] Document architecture (this spec)

### Phase 2: Kodos Integration (Future)

**Goal**: Enable generation creation on package operations

**Tasks**:
1. Research Kodos API for generation creation
2. Implement generation hook in chisel
3. Store registry snapshot in generation metadata
4. Add generation detection in chisel commands

**Estimated Effort**: 2-3 weeks

### Phase 3: Rollback Support (Future)

**Goal**: Enable registry restoration from generation snapshots

**Tasks**:
1. Implement registry snapshot loading
2. Detect generation changes
3. Auto-restore or prompt for registry sync
4. Validate registry consistency after restore

**Estimated Effort**: 2-3 weeks

### Phase 4: Testing & Validation (Future)

**Goal**: Ensure generation/registry integration works correctly

**Test Scenarios**:
- Single user: install → rollback → reinstall
- Multi-user: concurrent package operations
- Edge cases: corrupted registry, missing snapshot
- Performance: generation creation overhead

**Estimated Effort**: 2-3 weeks

### Phase 5: Documentation (Future)

**Tasks**:
1. Update USER-GUIDE.md with generation concepts
2. Create troubleshooting guide for rollbacks
3. Document per-user registry behavior
4. Add examples for multi-user setups

**Estimated Effort**: 1 week

---

## Design Decisions

### Decision 1: Registry Location

**Options Considered**:
1. Keep in user home: `$HOME/.local/share/chisel/registry.json`
2. Move to generation path: `/var/kod/generations/GEN_ID/registry.json`
3. Hybrid approach: mutable home + immutable snapshot in generation

**Decision**: Hybrid Approach (Option 3)

**Rationale**:
- Mutable registry in home enables package operations without touching immutable generation
- Immutable snapshot provides audit trail and rollback reference
- No locking needed because operations are generation-atomic
- Works seamlessly with existing chisel code

### Decision 2: Generation Creation Frequency

**Options Considered**:
1. Every operation creates generation (fine-grained history)
2. Batched operations, then user commits generation
3. Manual/periodic generation creation

**Decision**: Deferred - See Open Questions

**Rationale**: Depends on Kodos' performance and user expectations

### Decision 3: Automatic vs. Manual Registry Restoration

**Options Considered**:
1. Auto-restore from generation snapshot on rollback
2. Prompt user to confirm restoration
3. Manual restoration command only

**Decision**: Deferred - See Open Questions

**Rationale**: Depends on UX preferences and safety concerns

### Decision 4: Registry Format in Generations

**Options Considered**:
1. Plain JSON copy (readable, larger)
2. Compressed JSON.gz (smaller, less readable)
3. Both (redundancy)

**Decision**: Plain JSON (primary), optional compression for archives

**Rationale**: Readability for auditing, compression for long-term storage

---

## Open Questions

### Q1: Generation Creation Trigger

**Question**: Should chisel automatically trigger generation creation, or should the user manually trigger it?

**Options**:
- **A**: Automatic on every install/remove/upgrade
- **B**: Batched - operations queue until user commits with `chisel-user commit-generation`
- **C**: Manual - user runs `kodos create-generation` themselves

**Impact**: Affects user experience, storage usage, and rollback granularity

**Recommendation Request**: Clarify with Kodos team

### Q2: Generation Scope

**Question**: Should a generation be system-wide or per-user?

**Current Answer**: Per-user (users have their own generation chains)

**Clarification Needed**: 
- How do system-level packages interact with user generations?
- Can user packages in one generation see system packages from another?
- Should there be a system generation that all users inherit?

### Q3: Registry Restoration Behavior

**Question**: When a user switches generations, what happens to their registry?

**Options**:
- **A**: Auto-restore from generation snapshot
- **B**: Prompt user for confirmation
- **C**: Manual restoration command
- **D**: User manually re-runs `chisel-user sync`

**Impact**: Affects rollback experience and consistency

### Q4: Performance & Scalability

**Questions**:
- How large are typical registry.json files? (impact on generation storage)
- How many generations do users typically keep? (impact on storage)
- What's the performance impact of storing registry snapshot per generation?

**Investigation**: Benchmark with real-world chisel usage patterns

### Q5: Multi-User Package Sharing

**Question**: Should users be able to share packages across generations?

**Scenario**: User A and User B both install `vim`. Should it be stored once or twice?

**Impact**: Affects storage efficiency and dependency management

---

## Data Structures

### Registry Entry (Current, unchanged)

```go
type Package struct {
    Name         string    `json:"name"`
    Version      string    `json:"version"`
    Source       string    `json:"source"`      // "official" or "aur"
    Repository   string    `json:"repository"`  // "core", "extra", "aur"
    Files        []string  `json:"files"`
    Executables  []string  `json:"executables"`
    Dependencies []string  `json:"dependencies"`
    InstallDate  time.Time `json:"install_date"`
    UpdateDate   time.Time `json:"update_date"`
}
```

### Generation Metadata (New)

```go
type GenerationMetadata struct {
    ID              string            `json:"id"`
    CreatedAt       time.Time         `json:"created_at"`
    User            string            `json:"user"`
    Description     string            `json:"description"`  // optional: "Installed vim"
    RegistryPath    string            `json:"registry_path"`
    RegistryHash    string            `json:"registry_hash"`  // SHA256 for integrity
    Packages        map[string]*Package `json:"packages"`  // snapshot
    PreviousGenID   string            `json:"previous_gen_id"`
}
```

---

## Migration Path

### For Existing Systems (Not in scope for Phase 1)

When moving from current registry model to generation-based:

1. **Audit Current State**: Generate snapshot of current registry
2. **Create Initial Generation**: Treat current state as Generation 0
3. **Update Chisel**: Deploy new chisel with generation support
4. **Users Continue**: No changes to user workflows; automatic generation creation starts

---

## Risk Assessment

### Low Risk

- ✅ Per-user registry in home directory (already implemented)
- ✅ Generation snapshot storage (no impact on current functionality)

### Medium Risk

- ⚠️ Generation hook integration (need Kodos API stability)
- ⚠️ Automatic generation creation (performance TBD)

### High Risk

- ⚠️ Registry restoration on rollback (potential data loss if not careful)
- ⚠️ Multi-user generation interaction (complex coordination needed)

**Mitigation**: Comprehensive testing in Phase 4, clear documentation in Phase 5

---

## Success Criteria

### Phase 2 Success
- [ ] Generations created successfully on package operations
- [ ] Registry snapshot stored in generation metadata
- [ ] Generation creation doesn't break existing chisel functionality

### Phase 3 Success
- [ ] Users can restore registry from generation snapshot
- [ ] Rollback functionality works without data loss
- [ ] System detects and recovers from corrupted registry

### Phase 4 Success
- [ ] All test scenarios pass (single-user, multi-user, edge cases)
- [ ] Performance acceptable (< 1s overhead per operation)
- [ ] No data loss in any test scenario

### Phase 5 Success
- [ ] Documentation complete and accurate
- [ ] Users understand generation/registry relationship
- [ ] Support team can troubleshoot issues

---

## Future Enhancements (Out of Scope)

### Beyond Phase 5

1. **Registry Compression**: Compress old registry snapshots for storage efficiency
2. **Registry Diffing**: Show differences between generations
3. **Selective Rollback**: Rollback only specific packages (not entire generation)
4. **Registry Validation**: Verify registry consistency across generations
5. **Multi-generation Merge**: Combine packages from multiple generations
6. **Generation Tagging**: Tag important generations with user labels
7. **Registry Analytics**: Track package lifecycle, upgrade patterns, etc.

---

## References

- **Kodos Project**: https://github.com/kodos-prj/kodos
- **XDG Base Directory Spec**: https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
- **Chisel User Guide**: ./USER-GUIDE.md
- **Current Registry Implementation**: ./pkg/registry/registry.go

---

## Document History

| Date | Author | Status | Notes |
|------|--------|--------|-------|
| 2026-04-06 | OpenCode | DRAFT | Initial specification created |
| - | - | PENDING | Review and approval |
| - | - | PENDING | Architecture validation with Kodos team |

---

## Appendix: Example Generation Timeline

```
Generation 1 (Initial):
  └─ Packages: none
  └─ Created: 2026-04-06 10:00:00
  └─ User: alice

Generation 2 (Install vim):
  └─ Packages: vim (1.9.0-1)
  └─ Created: 2026-04-06 10:05:00
  └─ User: alice
  └─ Previous: Gen 1

Generation 3 (Install curl):
  └─ Packages: vim (1.9.0-1), curl (8.0.0-1)
  └─ Created: 2026-04-06 10:10:00
  └─ User: alice
  └─ Previous: Gen 2

Generation 4 (Upgrade vim):
  └─ Packages: vim (1.9.1-1), curl (8.0.0-1)
  └─ Created: 2026-04-06 10:15:00
  └─ User: alice
  └─ Previous: Gen 3

--- User Rollback to Gen 2 ---

Generation 5 (Restored from Gen 2):
  └─ Packages: vim (1.9.0-1)  [curl removed]
  └─ Created: 2026-04-06 10:20:00
  └─ User: alice
  └─ Previous: Gen 4 [rollback reference]
```

---

**END OF SPECIFICATION**
