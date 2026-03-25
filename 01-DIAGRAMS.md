# Packmgr System Diagrams

This document contains detailed visual diagrams to help understand the packmgr cross-distribution architecture, data flows, and component interactions.

**Version:** 4.0 (Cross-Distribution Architecture)  
**Date:** 2026-03-21

## Table of Contents

1. [System Architecture Diagrams](#system-architecture-diagrams)
2. [Cross-Distribution Architecture](#cross-distribution-architecture)
3. [Data Flow Diagrams](#data-flow-diagrams)
4. [Sequence Diagrams](#sequence-diagrams)
5. [Storage Layout Diagrams](#storage-layout-diagrams)
6. [Component Interaction Diagrams](#component-interaction-diagrams)

---

## 1. System Architecture Diagrams

### 1.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        USER INTERFACE                           │
├─────────────────────────────────────────────────────────────────┤
│                      CLI (Cobra + Go)                           │
│  install  │  remove  │  upgrade  │  search  │  query  │  list  │
└────────┬────────────────────────────────────────────────────────┘
         │
         │
┌────────▼────────────────────────────────────────────────────────┐
│                   APPLICATION LAYER (Go)                        │
├─────────────────────────────────────────────────────────────────┤
│  PackageManager                                                 │
│    - Install packages with dependencies                         │
│    - Remove packages safely                                     │
│    - Query and search packages                                  │
│    - Upgrade packages                                           │
├─────────────────────────────────────────────────────────────────┤
│  StorageManager                Registry                         │
│    - Extract packages          - Track installed pkgs           │
│    - Create symlinks           - Manage metadata                │
│    - Manage store              - Verify consistency             │
└────────┬───────────────────────────────────┬────────────────────┘
         │                                   │
         │                                   │
┌────────▼───────────────────────────────────▼────────────────────┐
│                INFRASTRUCTURE LAYER                             │
├─────────────────────────────────────────────────────────────────┤
│  ALPM Wrapper (go-alpm/v2)                                      │
│    - Dependency resolution                                      │
│    - Repository management                                      │
│    - Package metadata queries                                   │
├─────────────────────────────────────────────────────────────────┤
│  DownloadManager         │  Extractor         │  Verifier       │
│    - HTTP downloads      │  - Tar.zst extract │  - GPG verify   │
│    - Progress tracking   │  - File placement  │  - Checksums    │
└─────────────────────────────────────────────────────────────────┘
         │
         │
┌────────▼────────────────────────────────────────────────────────┐
│                      SYSTEM LAYER                               │
├─────────────────────────────────────────────────────────────────┤
│  libalpm.so  │  Filesystem  │  Network (HTTP/HTTPS)            │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Go Package Structure

```
packmgr-go/
│
├── cmd/packmgr/          [Entry Point]
│   └── main.go           - CLI initialization
│
├── internal/             [Core Implementation]
│   │
│   ├── alpm/             [ALPM Integration]
│   │   ├── wrapper.go    - ALPM handle management
│   │   ├── database.go   - Repository operations
│   │   └── package.go    - Package queries
│   │
│   ├── package/          [Package Management]
│   │   ├── manager.go    - Main package manager
│   │   ├── installer.go  - Installation logic
│   │   ├── remover.go    - Removal logic
│   │   └── registry.go   - Registry management
│   │
│   ├── storage/          [Storage Operations]
│   │   ├── store.go      - Package store
│   │   ├── symlink.go    - Symlink management
│   │   ├── extractor.go  - Archive extraction
│   │   └── manifest.go   - Store manifest
│   │
│   ├── download/         [Download Operations]
│   │   ├── fetcher.go    - HTTP downloads
│   │   ├── progress.go   - Progress bars
│   │   └── verifier.go   - Verification
│   │
│   ├── config/           [Configuration]
│   │   └── config.go     - Config management
│   │
│   └── cli/              [CLI Commands]
│       ├── root.go       - Root command
│       ├── install.go    - Install command
│       ├── remove.go     - Remove command
│       ├── search.go     - Search command
│       └── query.go      - Query commands
│
└── pkg/                  [Public API]
    └── registry/
        └── types.go      - Public types
```

### 1.3 Dependency Graph

```
┌──────────────────────────────────────────────────────────────┐
│ main.go                                                      │
└───────────────────────────┬──────────────────────────────────┘
                            │
                ┌───────────┴────────────┐
                │                        │
┌───────────────▼──────┐    ┌───────────▼────────────┐
│ CLI Commands         │    │ Config                 │
│ (install/remove/...) │    │                        │
└───────────────┬──────┘    └────────────────────────┘
                │
                │
┌───────────────▼─────────────────────────────────────────────┐
│ PackageManager                                              │
│   - Orchestrates package operations                         │
└───────┬──────────────────────────┬──────────────────────────┘
        │                          │
        │                          │
┌───────▼────────────┐   ┌─────────▼──────────────────────────┐
│ ALPM Wrapper       │   │ StorageManager                     │
│   - go-alpm/v2     │   │   - Store operations               │
│   - Dependencies   │   │   - Symlink management             │
└────────────────────┘   └─────────┬──────────────────────────┘
                                   │
                        ┌──────────┴───────────┐
                        │                      │
              ┌─────────▼────────┐   ┌────────▼──────────┐
              │ DownloadManager  │   │ Registry          │
              │   - HTTP fetch   │   │   - JSON storage  │
              └──────────────────┘   └───────────────────┘
```

---

## 2. Cross-Distribution Architecture

### 6.1 Cross-Distribution Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                     HOST DISTRIBUTIONS                           │
│  Ubuntu 22.04  │  Fedora 40  │  Debian 12  │  Arch Linux       │
│  glibc 2.35    │  glibc 2.39 │  glibc 2.36 │  glibc 2.39       │
└────────────────┬──────────────────────────────────────────────────┘
                 │
                 │  Packmgr runs on ANY distribution
                 │
┌────────────────▼──────────────────────────────────────────────────┐
│                        PACKMGR                                    │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  /kod/  (Isolated from host system)                        │  │
│  │  ├── db/           (Arch databases from mirrors)           │  │
│  │  ├── store/        (Arch packages + ALL dependencies)      │  │
│  │  ├── wrappers/     (Library isolation scripts)             │  │
│  │  └── cache/        (Downloaded packages)                   │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
                 │
                 │  Arch Linux packages + dependencies
                 │
┌────────────────▼──────────────────────────────────────────────────┐
│                  ARCH LINUX MIRRORS                               │
│  https://mirror.rackspace.com/archlinux/                         │
│  ├── core/os/x86_64/    (glibc, gcc-libs, etc.)                 │
│  └── extra/os/x86_64/   (vim, nginx, etc.)                      │
└──────────────────────────────────────────────────────────────────┘
```

**Key Principle:** Packmgr brings Arch packages to ANY Linux distribution by:
1. Installing ALL dependencies from Arch (complete isolation)
2. Using wrapper scripts to set library paths dynamically
3. Never mixing host and Arch libraries

### 6.2 Wrapper Script Execution Flow

```
┌──────────────────────────────────────────────────────────────────┐
│                  USER EXECUTES COMMAND                           │
│  user@ubuntu:~$ vim file.txt                                    │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  SHELL RESOLVES PATH                                             │
│  which vim → /usr/bin/vim (symlink)                              │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  SYMLINK RESOLUTION                                              │
│  /usr/bin/vim → /kod/wrappers/vim                                │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  WRAPPER SCRIPT EXECUTION                                        │
│  #!/bin/bash                                                     │
│  export LD_LIBRARY_PATH="/kod/store/vim/9.0/usr/lib:...         │
│                          /kod/store/glibc/2.39/usr/lib:...      │
│                          /kod/store/ncurses/6.4/usr/lib:..."    │
│  exec /kod/store/vim/9.0/usr/bin/vim "$@"                       │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  BINARY EXECUTION WITH ARCH LIBRARIES                            │
│  /kod/store/vim/9.0/usr/bin/vim                                  │
│  ├─ Loads: /kod/store/glibc/2.39/usr/lib/libc.so.6             │
│  ├─ Loads: /kod/store/ncurses/6.4/usr/lib/libncurses.so.6      │
│  └─ Loads: /kod/store/gcc-libs/13.2/usr/lib/libgcc_s.so.1      │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  SUCCESS!                                                        │
│  vim runs using Arch's glibc 2.39 on Ubuntu's glibc 2.35        │
│  No library conflicts, no crashes!                               │
└──────────────────────────────────────────────────────────────────┘
```

**Critical:** The wrapper ensures binaries ONLY load Arch libraries, never host libraries.

### 6.3 Database Sync Flow

```
┌──────────────────────────────────────────────────────────────────┐
│  USER SYNCS DATABASES                                            │
│  user@ubuntu:~$ sudo packmgr sync                                │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  READ MIRROR URL FROM CONFIG                                     │
│  mirror_url: "https://mirror.rackspace.com/archlinux"           │
│  repositories: ["core", "extra", "community"]                   │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  DOWNLOAD DATABASES FROM ARCH MIRRORS                            │
│  GET https://mirror.../core/os/x86_64/core.db                   │
│  GET https://mirror.../extra/os/x86_64/extra.db                 │
│  GET https://mirror.../community/os/x86_64/community.db         │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  SAVE TO /kod/db/                                                │
│  /kod/db/core.db       (100-150 KB, compressed)                 │
│  /kod/db/extra.db      (2-3 MB, compressed)                     │
│  /kod/db/community.db  (1-2 MB, compressed)                     │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  INITIALIZE ALPM WITH /kod ROOT                                  │
│  alpm_initialize("/kod", ...)                                    │
│  alpm_register_syncdb("core", ...)                              │
│  alpm_register_syncdb("extra", ...)                             │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  SUCCESS - READY FOR QUERIES                                     │
│  Can now search, query, and install packages                     │
└──────────────────────────────────────────────────────────────────┘
```

**Why explicit sync?**
- Packmgr is supplementary, not a replacement for host package manager
- User controls when to check for updates
- Reduces unnecessary network traffic

### 6.4 Complete Installation with Wrapper Generation

```
┌──────────────────────────────────────────────────────────────────┐
│  USER INSTALLS PACKAGE                                           │
│  user@ubuntu:~$ sudo packmgr install vim                         │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  DEPENDENCY RESOLUTION (ALPM with /kod root)                     │
│  Query: vim                                                      │
│  Resolved: [vim, glibc, gcc-libs, ncurses, gpm, vim-runtime]    │
│  Total: 12 packages (~200MB)                                    │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  DOWNLOAD PACKAGES FROM ARCH MIRRORS                             │
│  vim-9.0.1-1-x86_64.pkg.tar.zst                                 │
│  glibc-2.39-1-x86_64.pkg.tar.zst                                │
│  ... (10 more packages)                                          │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  EXTRACT TO /kod/store/                                          │
│  /kod/store/vim/9.0.1-1/usr/bin/vim                             │
│  /kod/store/glibc/2.39-1/usr/lib/libc.so.6                      │
│  /kod/store/ncurses/6.4-1/usr/lib/libncurses.so.6              │
│  ... (all files extracted)                                       │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  DISCOVER LIBRARY PATHS                                          │
│  Find all .so files in dependencies:                             │
│  - /kod/store/vim/9.0.1-1/usr/lib/*.so                          │
│  - /kod/store/glibc/2.39-1/usr/lib/*.so                         │
│  - /kod/store/ncurses/6.4-1/usr/lib/*.so                        │
│  - /kod/store/gcc-libs/13.2-1/usr/lib/*.so                      │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  GENERATE WRAPPER SCRIPT                                         │
│  /kod/wrappers/vim:                                              │
│  #!/bin/bash                                                     │
│  export LD_LIBRARY_PATH="/kod/store/vim/9.0.1-1/usr/lib:...     │
│                          /kod/store/glibc/2.39-1/usr/lib:...    │
│                          /kod/store/ncurses/6.4-1/usr/lib:..."  │
│  exec /kod/store/vim/9.0.1-1/usr/bin/vim "$@"                   │
│  chmod +x /kod/wrappers/vim                                      │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  CREATE SYMLINKS                                                 │
│  /usr/bin/vim → /kod/wrappers/vim                                │
│  /usr/bin/vimdiff → /kod/wrappers/vimdiff                        │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  UPDATE REGISTRY                                                 │
│  /kod/registry.json:                                             │
│  {                                                               │
│    "packages": [                                                 │
│      {"name": "vim", "version": "9.0.1-1", ...},                │
│      {"name": "glibc", "version": "2.39-1", ...},               │
│      ... (all installed packages)                                │
│    ]                                                             │
│  }                                                               │
└────────────────────────────┬─────────────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  SUCCESS! vim works on Ubuntu with Arch libraries                │
└──────────────────────────────────────────────────────────────────┘
```

---

## 3. Data Flow Diagrams

### 6.1 Package Installation Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    PACKAGE INSTALLATION                         │
└─────────────────────────────────────────────────────────────────┘

1. User Command
   $ packmgr install vim
                │
                ▼
2. CLI Parsing
   Parse command: install ["vim"]
                │
                ▼
3. Package Manager: Install()
   ┌─────────────────────────────────────────┐
   │ PackageManager.Install("vim")           │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
4. Dependency Resolution (ALPM)
   ┌─────────────────────────────────────────┐
   │ ALPM.ResolveDependencies("vim")         │
   │   Returns: [vim, gpm, vim-runtime]      │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
5. Conflict Detection
   ┌─────────────────────────────────────────┐
   │ Check for file conflicts                │
   │   - /usr/bin/vim exists? → Backup       │
   │   - Other package owns it? → Error      │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
6. Download Packages
   ┌─────────────────────────────────────────┐
   │ DownloadManager.Fetch(packages)         │
   │   - vim-9.0.1-1-x86_64.pkg.tar.zst      │
   │   - gpm-1.20.7-5-x86_64.pkg.tar.zst     │
   │   - vim-runtime-9.0.1-1-x86_64.pkg.tar  │
   │   [=====> Progress Bar ======>    ] 65% │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
7. Verify Signatures
   ┌─────────────────────────────────────────┐
   │ Verifier.CheckGPG(packages)             │
   │   ✓ vim-9.0.1-1 signature valid         │
   │   ✓ gpm-1.20.7-5 signature valid        │
   │   ✓ vim-runtime-9.0.1 signature valid   │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
8. Extract to Store
   ┌─────────────────────────────────────────┐
   │ Extractor.Extract(packages)             │
   │   /kod/store/vim/9.0.1-1/               │
   │   /kod/store/gpm/1.20.7-5/              │
   │   /kod/store/vim-runtime/9.0.1-1/       │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
9. Create Symlinks
   ┌─────────────────────────────────────────┐
   │ SymlinkManager.Create(packages)         │
   │   ln -s /kod/store/vim/9.0.1-1/usr/bin/vim │
   │         → /usr/bin/vim                  │
   │   ln -s /kod/store/gpm/1.20.7-5/usr/bin/gpm │
   │         → /usr/bin/gpm                  │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
10. Update Registry
   ┌─────────────────────────────────────────┐
   │ Registry.Add(packages)                  │
   │   registry.json updated:                │
   │   {                                     │
   │     "packages": [                       │
   │       {"name": "vim", "version": ...}   │
   │     ]                                   │
   │   }                                     │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
11. Success!
   ✓ vim-9.0.1-1 installed
   ✓ gpm-1.20.7-5 installed (dependency)
   ✓ vim-runtime-9.0.1-1 installed (dependency)
```

### 6.2 Package Removal Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    PACKAGE REMOVAL                              │
└─────────────────────────────────────────────────────────────────┘

1. User Command
   $ packmgr remove vim
                │
                ▼
2. CLI Parsing
   Parse command: remove ["vim"]
                │
                ▼
3. Package Manager: Remove()
   ┌─────────────────────────────────────────┐
   │ PackageManager.Remove("vim")            │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
4. Check Dependencies
   ┌─────────────────────────────────────────┐
   │ ALPM.CheckReverseDeps("vim")            │
   │   Returns: [neovim-plugin] (depends)    │
   │   → Error: Cannot remove (needed)       │
   │                                         │
   │ User: --force flag                      │
   │   → Continue removal                    │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
5. Detect Orphans
   ┌─────────────────────────────────────────┐
   │ DetectOrphans(vim)                      │
   │   - gpm: no longer needed → orphan      │
   │   - vim-runtime: no longer needed       │
   │   Ask user: Remove orphans? [Y/n]       │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
6. Remove Symlinks
   ┌─────────────────────────────────────────┐
   │ SymlinkManager.Remove(packages)         │
   │   rm /usr/bin/vim                       │
   │   rm /usr/bin/gpm                       │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
7. Update Registry
   ┌─────────────────────────────────────────┐
   │ Registry.Remove(packages)               │
   │   registry.json updated:                │
   │   - vim removed                         │
   │   - gpm removed                         │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
8. Cleanup Store? (Optional)
   ┌─────────────────────────────────────────┐
   │ If --cleanup flag:                      │
   │   StorageManager.Delete(packages)       │
   │   rm -rf /kod/store/vim/9.0.1-1/        │
   │                                         │
   │ If NOT --cleanup (default):             │
   │   Keep in store for potential reinstall │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
9. Success!
   ✓ vim removed
   ✓ gpm removed (orphan)
   ✓ vim-runtime removed (orphan)
   Note: Packages kept in store (use cleanup to remove)
```

### 6.3 Search & Query Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    SEARCH PACKAGES                              │
└─────────────────────────────────────────────────────────────────┘

1. User Command
   $ packmgr search python
                │
                ▼
2. ALPM Search
   ┌─────────────────────────────────────────┐
   │ ALPM.SearchDatabases("python")          │
   │   Query: core, extra, community         │
   │   Match: name OR description            │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
3. Results Processing
   ┌─────────────────────────────────────────┐
   │ Sort and format results:                │
   │                                         │
   │ core/python 3.11.6-1                    │
   │     The Python programming language     │
   │                                         │
   │ extra/python-pip 23.3-1                 │
   │     PyPA tool for installing packages   │
   │                                         │
   │ extra/python-setuptools 68.2.2-1        │
   │     Build and distribute Python pkgs    │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
4. Display Results
   Results displayed to user


┌─────────────────────────────────────────────────────────────────┐
│                    QUERY PACKAGE INFO                           │
└─────────────────────────────────────────────────────────────────┘

1. User Command
   $ packmgr info vim
                │
                ▼
2. Check Registry First
   ┌─────────────────────────────────────────┐
   │ Registry.GetPackage("vim")              │
   │   Found? → Get local info               │
   │   Not found? → Query ALPM               │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
3. ALPM Query
   ┌─────────────────────────────────────────┐
   │ ALPM.GetPackageInfo("vim")              │
   │   - Name, version, description          │
   │   - Dependencies                        │
   │   - Size, install date                  │
   │   - Repository                          │
   └───────────────────┬─────────────────────┘
                       │
                       ▼
4. Display Info
   Name         : vim
   Version      : 9.0.1-1
   Repository   : extra
   Installed    : Yes
   Install Date : 2026-03-21 10:00:00
   Size         : 3.0 MB
   Dependencies : gpm, vim-runtime
   Description  : Vi Improved text editor
```

---

## 4. Sequence Diagrams

### 6.1 Installation Sequence

```
User         CLI        PkgMgr      ALPM       Download    Storage     Registry
 │           │           │           │            │           │           │
 │  install  │           │           │            │           │           │
 │──────────>│           │           │            │           │           │
 │           │  Install()│           │            │           │           │
 │           │──────────>│           │            │           │           │
 │           │           │ Resolve() │            │           │           │
 │           │           │──────────>│            │           │           │
 │           │           │  deps[]   │            │           │           │
 │           │           │<──────────│            │           │           │
 │           │           │  Fetch()  │            │           │           │
 │           │           │───────────────────────>│           │           │
 │           │           │           │   pkgs[]   │           │           │
 │           │           │<───────────────────────│           │           │
 │           │           │  Extract()│            │           │           │
 │           │           │───────────────────────────────────>│           │
 │           │           │           │            │   done    │           │
 │           │           │<───────────────────────────────────│           │
 │           │           │  CreateSymlinks()      │           │           │
 │           │           │───────────────────────────────────>│           │
 │           │           │           │            │   done    │           │
 │           │           │<───────────────────────────────────│           │
 │           │           │  Update() │            │           │           │
 │           │           │───────────────────────────────────────────────>│
 │           │  Success  │           │            │           │           │
 │<──────────│<──────────│           │            │           │           │
```

### 6.2 Removal Sequence

```
User         CLI        PkgMgr      ALPM       Storage     Registry
 │           │           │           │           │           │
 │  remove   │           │           │           │           │
 │──────────>│           │           │           │           │
 │           │  Remove() │           │           │           │
 │           │──────────>│           │           │           │
 │           │           │CheckDeps()│           │           │
 │           │           │──────────>│           │           │
 │           │           │  OK/Error │           │           │
 │           │           │<──────────│           │           │
 │           │  Confirm? │           │           │           │
 │<──────────│<──────────│           │           │           │
 │    Yes    │           │           │           │           │
 │──────────>│──────────>│           │           │           │
 │           │           │  RemoveSymlinks()     │           │
 │           │           │───────────────────────>│           │
 │           │           │           │   done    │           │
 │           │           │<───────────────────────│           │
 │           │           │  Update() │           │           │
 │           │           │───────────────────────────────────>│
 │           │  Success  │           │           │           │
 │<──────────│<──────────│           │           │           │
```

---

## 5. Storage Layout Diagrams

### 5.1 Directory Structure (v4.0 - Cross-Distribution)

```
/kod/                                    [Root Directory - Isolated from host]
│
├── db/                                  [Arch Package Databases - NEW]
│   ├── core.db                          [Core repository database]
│   ├── extra.db                         [Extra repository database]
│   └── community.db                     [Community repository database]
│
├── store/                               [Package Store - All Arch packages]
│   │
│   ├── glibc/                           [System library from Arch - NEW]
│   │   └── 2.39-1/
│   │       └── usr/
│   │           └── lib/
│   │               ├── libc.so.6
│   │               ├── libm.so.6
│   │               └── ld-linux-x86-64.so.2
│   │
│   ├── gcc-libs/                        [Compiler runtime from Arch - NEW]
│   │   └── 13.2-1/
│   │       └── usr/
│   │           └── lib/
│   │               ├── libgcc_s.so.1
│   │               └── libstdc++.so.6
│   │
│   ├── ncurses/                         [Terminal library from Arch]
│   │   └── 6.4-1/
│   │       └── usr/
│   │           └── lib/
│   │               └── libncurses.so.6
│   │
│   ├── bash/                            [Package: bash]
│   │   └── 5.2.26-1/                    [Version: 5.2.26-1]
│   │       ├── usr/
│   │       │   ├── bin/
│   │       │   │   └── bash             [Actual binary]
│   │       │   └── share/
│   │       │       ├── man/man1/
│   │       │       │   └── bash.1.gz
│   │       │       └── doc/bash/
│   │       └── etc/
│   │           └── bash.bashrc
│   │
│   ├── vim/                             [Package: vim]
│   │   ├── 9.0.1-1/                     [Current version]
│   │   │   ├── usr/
│   │   │   │   ├── bin/vim
│   │   │   │   ├── lib/                 [vim's libraries]
│   │   │   │   │   └── *.so
│   │   │   │   └── share/vim/
│   │   │   └── etc/
│   │   │       └── vimrc
│   │   └── 9.0.0-2/                     [Old version (kept)]
│   │       └── usr/...
│   │
│   └── nginx/
│       └── 1.24.0-1/
│           ├── usr/
│           │   ├── bin/nginx
│           │   └── share/nginx/
│           └── etc/
│               └── nginx/
│                   └── nginx.conf
│
├── wrappers/                            [Wrapper Scripts - NEW]
│   ├── bash                             [Sets LD_LIBRARY_PATH for bash]
│   ├── vim                              [Sets LD_LIBRARY_PATH for vim]
│   └── nginx                            [Sets LD_LIBRARY_PATH for nginx]
│
├── cache/                               [Download Cache]
│   ├── bash-5.2.26-1-x86_64.pkg.tar.zst
│   ├── vim-9.0.1-1-x86_64.pkg.tar.zst
│   ├── glibc-2.39-1-x86_64.pkg.tar.zst
│   └── nginx-1.24.0-1-x86_64.pkg.tar.zst
│
├── registry.json                        [Installed Packages Registry]
│
└── config.json                          [Configuration - JSON format]
```

**NEW in v4.0:**
- `/kod/db/` - Synced Arch databases for queries
- `/kod/wrappers/` - Generated wrapper scripts for library isolation
- System libraries (glibc, gcc-libs, ncurses) stored alongside packages
- Config in JSON format (not YAML)

### 5.2 Wrapper + Symlink Structure (v4.0)

```
System Filesystem           Wrapper Script              Package Store
─────────────────           ──────────────              ─────────────

/usr/bin/bash  ──────>  /kod/wrappers/bash  ──────>  /kod/store/bash/5.2.26-1/usr/bin/bash
   (symlink)             (shell script)                 (actual binary)
                         Sets LD_LIBRARY_PATH

/usr/bin/vim   ──────>  /kod/wrappers/vim   ──────>  /kod/store/vim/9.0.1-1/usr/bin/vim


Wrapper Script Example (/kod/wrappers/vim):
────────────────────────────────────────────
#!/bin/bash
export LD_LIBRARY_PATH="/kod/store/vim/9.0.1-1/usr/lib:\
/kod/store/glibc/2.39-1/usr/lib:\
/kod/store/ncurses/6.4-1/usr/lib:\
/kod/store/gcc-libs/13.2-1/usr/lib:$LD_LIBRARY_PATH"
exec /kod/store/vim/9.0.1-1/usr/bin/vim "$@"


Library Loading Flow:
─────────────────────
User types: vim file.txt
    ↓
/usr/bin/vim (symlink) → /kod/wrappers/vim (wrapper)
    ↓
Wrapper sets LD_LIBRARY_PATH → Arch library directories
    ↓
Wrapper execs /kod/store/vim/9.0.1-1/usr/bin/vim
    ↓
Binary loads libraries from /kod/store/ (NOT /usr/lib/)
    ↓
Success! Arch binary runs with Arch libraries on ANY distro
```

**Why Two-Tier System?**
- **Symlinks alone fail:** Binary would load host libraries (incompatible glibc versions)
- **Wrappers solve this:** Set `LD_LIBRARY_PATH` before execution
- **Complete isolation:** Arch binaries only see Arch libraries

### 5.3 Registry Structure

```json
{
  "version": "1.0",
  "updated": "2026-03-21T16:45:00Z",
  "packages": [
    {
      "name": "bash",
      "version": "5.2.26-1",
      "repository": "core",
      "installed": "2026-03-21T10:00:00Z",
      "install_reason": "explicit",
      "size": 1572864,
      "files": [
        "/usr/bin/bash",
        "/usr/share/man/man1/bash.1.gz",
        "/etc/bash.bashrc"
      ],
      "dependencies": ["glibc", "readline", "ncurses"]
    },
    {
      "name": "vim",
      "version": "9.0.1-1",
      "repository": "extra",
      "installed": "2026-03-21T10:05:00Z",
      "install_reason": "explicit",
      "size": 3145728,
      "files": [
        "/usr/bin/vim",
        "/etc/vimrc",
        "/usr/share/vim/..."
      ],
      "dependencies": ["gpm", "vim-runtime"]
    }
  ],
  "total_packages": 2,
  "total_size": 4718592
}
```

---

## 6. Component Interaction Diagrams

### 6.1 Component Relationships

```
┌────────────────────────────────────────────────────────────────┐
│                        PackageManager                          │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                     Public API                           │  │
│  │  - Install(packages []string)                            │  │
│  │  - Remove(packages []string)                             │  │
│  │  - Upgrade(packages []string)                            │  │
│  │  - Search(query string) []Package                        │  │
│  │  - Query(name string) Package                            │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  Dependencies:                                                  │
│  ┌────────────┐  ┌─────────────┐  ┌──────────┐               │
│  │ ALPMWrapper│  │ Storage     │  │ Registry │               │
│  └────────────┘  └─────────────┘  └──────────┘               │
└─────────────────────────────────────────────────────────────────┘


┌────────────────────────────────────────────────────────────────┐
│                        ALPMWrapper                             │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  - Initialize(root, dbpath string)                       │  │
│  │  - ResolveDependencies(pkg string) []string              │  │
│  │  - SearchDatabases(query string) []Package               │  │
│  │  - GetPackage(name string) Package                       │  │
│  │  - CheckConflicts(pkg string) error                      │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  Uses: go-alpm/v2 (github.com/Jguer/go-alpm/v2)               │
└─────────────────────────────────────────────────────────────────┘


┌────────────────────────────────────────────────────────────────┐
│                      StorageManager                            │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  - ExtractPackage(pkgPath, destPath string) error        │  │
│  │  - CreateSymlink(target, link string) error              │  │
│  │  - RemoveSymlink(link string) error                      │  │
│  │  - CreateSymlinks(pkg Package) error                     │  │
│  │  - RemoveSymlinks(pkg Package) error                     │  │
│  │  - VerifyStore() error                                   │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  Components:                                                    │
│  ┌────────────┐  ┌─────────────┐                              │
│  │ Extractor  │  │ Symlinker   │                              │
│  └────────────┘  └─────────────┘                              │
└─────────────────────────────────────────────────────────────────┘


┌────────────────────────────────────────────────────────────────┐
│                      DownloadManager                           │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  - FetchPackages(pkgs []string) ([]string, error)        │  │
│  │  - VerifySignature(pkgPath string) error                 │  │
│  │  - ShowProgress(current, total int)                      │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  Components:                                                    │
│  ┌────────────┐  ┌─────────────┐                              │
│  │ HTTPClient │  │ Verifier    │                              │
│  └────────────┘  └─────────────┘                              │
└─────────────────────────────────────────────────────────────────┘


┌────────────────────────────────────────────────────────────────┐
│                          Registry                              │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  - Load() error                                          │  │
│  │  - Save() error                                          │  │
│  │  - AddPackage(pkg Package) error                        │  │
│  │  - RemovePackage(name string) error                     │  │
│  │  - GetPackage(name string) (Package, error)             │  │
│  │  - ListPackages() []Package                             │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  Storage: registry.json (JSON format)                          │
└─────────────────────────────────────────────────────────────────┘
```

### 6.2 Error Handling Flow

```
┌────────────────────────────────────────────────────────────────┐
│                      Error Propagation                         │
└────────────────────────────────────────────────────────────────┘

Installation Error Handling:
─────────────────────────────

PackageManager.Install()
    │
    ├─> ALPM error?
    │   └─> Return: "failed to resolve dependencies: ..."
    │
    ├─> Download error?
    │   └─> Return: "failed to download package: ..."
    │
    ├─> Extraction error?
    │   └─> Rollback: Remove partial extraction
    │   └─> Return: "failed to extract package: ..."
    │
    ├─> Symlink error?
    │   └─> Rollback: Remove created symlinks
    │   └─> Rollback: Remove extracted files
    │   └─> Return: "failed to create symlinks: ..."
    │
    └─> Registry error?
        └─> Warning: "Package installed but registry not updated"
        └─> Return: success with warning


Rollback Strategy:
──────────────────

Error at Step N
    │
    ├─> Undo Step N-1
    ├─> Undo Step N-2
    ├─> ...
    └─> Undo Step 1
    │
    └─> Return to clean state


Example:
────────

Install vim → Error creating symlink
    │
    ├─> Remove partial symlinks
    ├─> Remove extracted files from /kod/store/vim/
    ├─> Don't update registry
    └─> Return error to user
```

---

## 6. State Diagrams

### 6.1 Package States

```
┌───────────────────────────────────────────────────────────────┐
│                      Package Lifecycle                        │
└───────────────────────────────────────────────────────────────┘

  [Not Installed]
        │
        │ install
        ▼
  [Downloading]
        │
        │ downloaded
        ▼
  [Verifying]
        │
        │ verified
        ▼
  [Extracting]
        │
        │ extracted
        ▼
  [Creating Symlinks]
        │
        │ symlinks created
        ▼
  [Installed] ◄──────────────┐
        │                    │
        │ upgrade            │ verify
        ▼                    │
  [Upgrading] ───────────────┘
        │
        │ remove
        ▼
  [Removing Symlinks]
        │
        │ symlinks removed
        ▼
  [Removed from Registry]
        │
        │ cleanup
        ▼
  [Deleted from Store]
        │
        ▼
  [Not Installed]
```

### 6.2 Registry States

```
  [Empty Registry]
        │
        │ first install
        ▼
  [Registry Exists]
        │
        │ ┌────────────────┐
        │ │ package added  │
        ▼ ▼                │
  [Updated Registry] ──────┘
        │
        │ sync
        ▼
  [Synchronized]
```

---

## Document Control

**Revision History:**

| Version | Date | Changes |
|---------|------|---------|
| 3.0 | 2026-03-21 | Simplified for Go implementation, removed generations |
| 2.0 | 2026-03-21 | Complex generation-based design |
| 1.0 | 2026-01-01 | Initial diagrams |

---

*End of Diagrams Document*
