# Chisel2 Research Documentation Index

## Overview

This directory contains comprehensive research on chisel2's registry.json design and compatibility with Kodos architecture patterns. All documents are generated from detailed code analysis and source code examination.

## Documentation Files

### Core Research Documents

#### 1. **REGISTRY-ANALYSIS.md** (933 lines, 31KB)
**The comprehensive research document - START HERE**

Complete analysis answering all research questions with code examples, architectural comparisons, and recommendations.

**Contents:**
- Executive Summary
- Part 1: Current Registry.json Usage (detailed breakdown)
  - How registry is used
  - Read/write frequency analysis
  - Locking and concurrency issues
  - Persistence requirements
- Part 2: What Happens If Lost/Corrupted
  - Package functionality
  - Recovery options
  - Current behavior on corruption
- Part 3: Current Design and Multi-User/Multi-System
  - Multiple users support
  - System upgrades and reinstalls
  - Atomicity of operations
- Part 4: Kodos Generation Architecture
  - Definition of "generation"
  - Immutability and rollback
  - Package metadata storage
- Part 5: Compatibility Analysis
  - Proposed architecture change
  - Critical incompatibilities
  - Feasibility assessment
  - Recommendations
- Appendix A: Code Locations
- Appendix B: Recovery Procedures

**Best for:** Deep understanding, architectural decisions, implementation planning

---

#### 2. **REGISTRY-QUICK-REFERENCE.md** (236 lines, 6.7KB)
**Quick lookup guide - USE FOR REFERENCE**

Single-page reference for developers implementing registry improvements.

**Contents:**
- Current usage table
- Data structure definition
- Command lifecycle diagrams
- Concurrency issues (with examples)
- Loss/corruption scenarios
- Multi-user support matrix
- Chisel vs Kodos comparison table
- Recommended improvements
- Code locations
- Recovery procedures
- Design rationale

**Best for:** Daily reference, quick lookups, teaching new developers

---

### Related Documentation

#### 3. **ARCHITECTURE-ANALYSIS.md** (769 lines, 25KB)
Existing comprehensive codebase architecture documentation covering dependency resolution, database schema, package metadata, and command flows.

#### 4. **ARCHITECTURE-CODE-FLOW.md** (744 lines, 22KB)
Existing detailed code flow documentation for install, remove, upgrade, and cleanup commands with registry interaction points.

#### 5. **ARCHITECTURE-QUICK-REF.md** (267 lines, 7.2KB)
Existing quick reference for architecture with component overview and flow diagrams.

---

## Key Findings at a Glance

### Current State
- **Registry Format:** JSON map of Package structs
- **Location:** `/kod/registry.json` (configurable)
- **I/O Pattern:** Read-once, modify-in-memory, write-once per command
- **Thread Safety:** In-process (sync.RWMutex), NO inter-process locking
- **Frequency:** Used on every package operation
- **Persistence:** Essential - survives reboots and upgrades

### Critical Issues
- **Concurrency:** No inter-process locking (race conditions in multi-user scenarios)
- **Atomicity:** No all-or-nothing guarantees (partial failures possible)
- **Recovery:** No automatic recovery if lost/corrupted
- **Multi-user:** Unsupported (single admin user assumption)

### Kodos Differences
| Aspect | Chisel | Kodos |
|--------|--------|-------|
| Generations | None | Btrfs snapshots |
| Registry Location | `/kod/registry.json` | `/var/kod/generations/N/` |
| Mutability | Mutable | Immutable per-generation |
| Atomicity | Per-invocation | Per-generation |
| Rollback | Manual | Automatic (reboot) |
| Multi-user | Unsupported | Supported |

### Recommendations

**SHORT-TERM (1-2 weeks, ~100-200 lines):**
1. Add file-level locking (fcntl) for multi-user support
2. Add corruption detection (backup before write)
3. Improve error handling (atomic writes with validation)

**DO NOT** move registry to `/var/kod/` without:
- Full generation system implementation
- Immutability enforcement
- Atomic generation switching
- Complete architectural redesign

**LONG-TERM (if needed):**
- Design generation system (weeks of work)
- Implement generation creation/switching
- Add rollback support
- Then `/var/kod/generations/N/registry.json` becomes appropriate

---

## Code Reference

### Registry Implementation
- **Main implementation:** `/pkg/registry/registry.go` (163 lines)
  - Registry struct and methods
  - Load/Save operations
  - Package operations (Add/Remove/Get)
  - Type definitions

### Registry Usage
- **Install command:** `/internal/cli/install.go:483-528`
  - Opens registry, batches package additions, single save
- **Remove command:** `/internal/cli/remove.go:72-176`
  - Opens registry, removes packages, single save
- **Upgrade command:** `/internal/cli/upgrade.go:102-170`
  - Multiple read/write cycles (potential issues)
- **List command:** `/internal/cli/list.go:29`
  - Opens registry, reads packages, no writes
- **Cleanup command:** `/internal/cli/cleanup.go:118`
  - Opens registry, reads for state, no writes

### Configuration
- **Config struct:** `/pkg/config/config.go:23-78`
- **Registry path defaults:** Lines 87, 229, 292
- **Config loading:** Lines 80-127

### Tests
- **Registry tests:** `/pkg/registry/registry_test.go`
- **AUR registry tests:** `/pkg/registry/registry_aur_test.go`
- **Integration tests:** `/integration/full_workflow_test.go`

---

## Kodos Research

### Sources Examined
- **Kodos repository:** https://github.com/kodos-prj/kodos
- **Generation functions:** `src/kod/core.py:get_max_generation()`, `create_next_generation()`
- **Metadata storage:** `src/kod/core.py:store_packages_services()`, `load_package_lock()`
- **Generation structure:** `/kod/generations/N/` with rootfs, installed_packages, enabled_services, packages.lock

### Key Insights
- Generations are **immutable** once created
- Complete system state included (filesystem + metadata)
- Bootable via kernel parameter selection
- Rollback by selecting previous generation
- Metadata **per-generation**, not global

---

## How to Use These Documents

### For Decision Makers
1. Read: REGISTRY-ANALYSIS.md → Part 5 (Compatibility Analysis)
2. Review: REGISTRY-QUICK-REFERENCE.md → Recommendations
3. Decision: Move registry to /var/kod? → NO (with architectural changes)

### For Developers Implementing Improvements
1. Read: REGISTRY-QUICK-REFERENCE.md (overview)
2. Reference: Code Locations section
3. Implement: File-level locking (recommended short-term improvement)
4. Test: Using provided test scenarios

### For System Administrators
1. Read: REGISTRY-QUICK-REFERENCE.md → Recovery Procedures
2. Know: Packages work even if registry is lost
3. Action: Regular backups of `/kod/registry.json`
4. Reference: Multi-user warnings if applicable

### For Architecture Reviews
1. Read: REGISTRY-ANALYSIS.md (all parts)
2. Review: ARCHITECTURE-ANALYSIS.md (context)
3. Understand: Current design rationale
4. Plan: Future improvements (if needed)

---

## Document Conventions

### Code References
- File paths: `/pkg/registry/registry.go`
- Line numbers: `registry.go:14`
- Functions: `registry.Load()`
- Types: `Registry struct`

### Terminology
- **Registry:** `/kod/registry.json` file + in-memory map
- **Generation:** Snapshot of system state (Kodos concept)
- **Atomicity:** All-or-nothing operation guarantee
- **Race condition:** Concurrent access problem
- **TOCTOU:** Time-of-Check-Time-of-Use race condition

### Icons/Formatting
- ✓ Supported/Works
- ✗ Not supported/Fails
- **Bold:** Important concepts
- `Code`: File paths, type names, functions
- `> Command`: Shell commands

---

## Summary

**Research Question:** Should registry.json be moved to `/var/kod` following Kodos architecture?

**Answer:** NO - Not recommended without broader architectural changes

**Reason:** Chisel has fundamentally different architecture than Kodos:
- No generation concept
- Per-invocation atomicity vs per-generation
- Single global registry vs per-generation metadata
- Moving path provides no benefits alone

**Recommended Action:** Focus on robustness improvements (locking, error handling) in current architecture.

---

## Additional Resources

- **Kodos Project:** https://github.com/kodos-prj/kodos
- **Chisel2 GitHub:** https://github.com/kodos-prj/chisel
- **Arch Linux Package Format:** https://wiki.archlinux.org/title/Arch_packaging_standards
- **Btrfs Documentation:** https://btrfs.readthedocs.io/

---

## Document Generation

- **Generated:** April 6, 2026
- **Analysis Date:** April 6, 2026
- **Kodos Repository:** Latest (commit analyzed from main branch)
- **Chisel2 Repository:** Work-in-progress (analyzed from /home/abuss/Work/devel/chisel2)
- **Research Time:** Comprehensive code review + cross-project analysis

---

## Contact & Questions

For questions about these findings:
1. Review the appropriate document section
2. Check the code references provided
3. Refer to the recovery procedures for operational issues
4. Consult ARCHITECTURE-ANALYSIS.md for broader context

---

**Last Updated:** April 6, 2026
**Status:** Complete - All research questions answered
