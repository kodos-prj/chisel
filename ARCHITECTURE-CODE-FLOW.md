# Chisel Architecture - Code Flow Diagrams

## Full Installation Flow

```
CLI: chisel install bash
  ↓
cmd/chisel/main.go:317
  handleInstall(args []string)
  ├─ loadConfig() → pkg/config/config.go:152
  ├─ NewInstallCommandWithSymlinkDir(cfg, symlinkDir)
  └─ cmd.Run(args)
    ↓
internal/cli/install.go:56
  InstallCommand.Run(args []string)
  ├─ Parse CLI flags (--no-deps, --no-extract, --no-symlink, --force)
  ├─ pkgNames ← remaining args
  └─ Main processing:
    
    [STAGE 1: DEPENDENCY RESOLUTION]
    ├─ client := alpm.NewClient(cfg.AlpmRoot, cfg.AlpmDBPath)
    │  └─ pkg/alpm/alpm.go:58 → NewGoClient(dbPath, arch)
    │
    ├─ client.RegisterAllSyncDBs(repos) → pkg/alpm/db.go:38
    │  └─ For each repo: LoadCachedDatabase(repo) → pkg/alpm/parse.go:252
    │     ├─ Read /kod/db/sync/{repo}.db
    │     ├─ Decompress gzip
    │     └─ Parse tar: parsePackageDatabase() → pkg/alpm/parse.go:18
    │
    ├─ resolveDependenciesWithMap(client, pkgNames, skipDeps)
    │  └─ internal/cli/install.go:465
    │     └─ For each pkgName:
    │        ├─ client.ResolveDependencies(pkgName)
    │        │  └─ pkg/alpm/db.go:130
    │        │     ├─ Find package: Cache.GetPackage(name)
    │        │     └─ resolveDepsRecursive() → pkg/alpm/db.go:150
    │        │        ├─ Cycle detection: check visiting[pkg]
    │        │        ├─ For each dependency:
    │        │        │  ├─ ParseDependency(depStr)
    │        │        │  ├─ Cache.GetPackage(depName, arch)
    │        │        │  ├─ Fall back: Cache.GetProvidingPackages(depName)
    │        │        │  ├─ CheckVersionConstraint(version, constraint)
    │        │        │  │  └─ VerCmp() → pkg/alpm/version.go:9
    │        │        │  └─ Recurse: resolveDepsRecursive()
    │        │        └─ Append to result[]
    │        │
    │        ├─ Skip if installed (registry check)
    │        └─ Append to toInstall: []download.PackageInfo
    │
    │     Returns: (toInstall, depMap, nil)
    │
    │ toInstall = [ {bash, 5.3.9-1, core}, {glibc, 2.37-1, core}, ... ]
    │ depMap = { bash: [glibc, ncurses, ...], ... }
    
    [STAGE 2: DOWNLOAD]
    ├─ downloader := download.NewDownloader(
    │    mirrorURL, cachePath, arch, maxConcurrent, timeout)
    │
    ├─ downloader.DownloadPackages(toInstall)
    │  └─ pkg/download/download.go:103
    │     ├─ Create semaphore(maxConcurrent)
    │     └─ For each package (concurrent):
    │        ├─ Construct URL: {mirror}/{repo}/os/{arch}/{name}-{version}-{arch}.pkg.tar.zst
    │        ├─ HTTP GET (with timeout)
    │        ├─ Create /kod/cache/{name}-{version}-{arch}.pkg.tar.zst.tmp
    │        ├─ io.Copy(tmpFile, response.Body)
    │        ├─ tmpFile.Close()
    │        ├─ os.Rename(tmpPath, finalPath) [atomic]
    │        └─ results[name] = finalPath
    │
    │ results = {
    │   bash: /kod/cache/bash-5.3.9-1-x86_64.pkg.tar.zst,
    │   glibc: /kod/cache/glibc-2.37-1-x86_64.pkg.tar.zst,
    │   ...
    │ }
    
    [STAGE 3: EXTRACT]
    ├─ storeManager := store.NewStore(storeRoot)
    │  └─ pkg/store/store.go:27
    │
    ├─ For each downloaded package:
    │  ├─ storeManager.ExtractPackage(cachePath, pkgName, version)
    │  │  └─ pkg/store/store.go:50
    │  │     ├─ destDir ← /kod/store/{pkgName}/{version}
    │  │     └─ extractor.ExtractPackage(pkgPath, destDir)
    │  │        └─ pkg/extract/extract.go:39
    │  │           ├─ os.Open(pkgPath)
    │  │           ├─ zstd.NewReader() [decompress]
    │  │           ├─ tar.NewReader()
    │  │           ├─ For each tar entry:
    │  │           │  ├─ If TypeReg: create file, io.Copy(), chmod
    │  │           │  ├─ If TypeDir: os.MkdirAll()
    │  │           │  ├─ If TypeSymlink: os.Symlink()
    │  │           │  └─ If TypeLink: os.Link()
    │  │           └─ Return: []ExtractedFile
    │  │
    │  ├─ extractedFilesMap[pkgName][version] ← ExtractedFile list
    │  └─ storeManager.SetLatestVersion(pkgName, version)
    │     └─ Create/update /kod/store/{pkgName}/current → {version}
    
    [STAGE 4: SYMLINK CREATION]
    ├─ If NOT --no-symlink AND symlinkDir != "":
    │  └─ For each package in toInstall:
    │     ├─ Get extractedFilesMap[pkgName][version]
    │     └─ For each extracted file:
    │        ├─ Skip metadata (.PKGINFO, .BUILDINFO, .MTREE, .INSTALL)
    │        ├─ If file is extracted symlink:
    │        │  └─ targetPath ← storage + target
    │        ├─ Else if file in usr/bin/* or usr/sbin/*:
    │        │  └─ targetPath ← /kod/wrappers/{fileName}
    │        ├─ Else:
    │        │  └─ targetPath ← /kod/store/{pkg}/{version}/{file}
    │        │
    │        ├─ symlinkPath ← {symlinkDir}/{file}
    │        ├─ Check if already exists:
    │        │  ├─ If points to correct location: skip
    │        │  ├─ If points elsewhere: skip (unless --force)
    │        │  └─ If regular file: skip
    │        ├─ os.MkdirAll(parent)
    │        ├─ os.Symlink(targetPath, symlinkPath)
    │        └─ symlinkCount++
    
    [STAGE 5: WRAPPER GENERATION]
    ├─ wrapperGen := wrapper.NewGenerator(
    │    storeRoot, wrapperDir, symlinkRoot)
    │
    ├─ For each package in toInstall:
    │  ├─ wrapperGen.DiscoverLibraries(pkgName, version)
    │  │  └─ Find all .so files in /kod/store/{pkg}/{version}/usr/lib/*
    │  │
    │  └─ For each executable (usr/bin/*, usr/sbin/*):
    │     ├─ wrapperGen.GenerateWrapperWithDeps(
    │     │    cmdName, pkgName, version, libDirs, dependencies)
    │     │
    │     └─ Create shell script in /kod/wrappers/{cmdName}:
    │        ├─ Set LD_LIBRARY_PATH to /kod/store/{dep}/{ver}/usr/lib/...
    │        ├─ Exec /kod/store/{pkgName}/{version}/{execPath}
    │        └─ Create symlink: /{symlinkRoot}/usr/bin/{cmdName} → wrapper
    
    [STAGE 6: REGISTRY UPDATE]
    ├─ reg := registry.NewRegistry(registryPath)
    │  └─ pkg/registry/registry.go:35
    │     └─ Load existing /kod/registry.json (if exists)
    │
    ├─ For each package in toInstall:
    │  ├─ regPkg := &registry.Package{
    │  │    Name: pkgInfo.Name,
    │  │    Version: pkgInfo.Version,
    │  │    Files: extractedFilesMap[name][ver].AllFiles,
    │  │    Executables: extractedFilesMap[name][ver].Executables,
    │  │    Dependencies: depMap[name],
    │  │    InstallDate: time.Now().Format(RFC3339),
    │  │  }
    │  ├─ reg.AddPackage(regPkg)
    │  └─ reg.Save() → /kod/registry.json (JSON marshal)
    │
    └─ Return success message
```

---

## Dependency Resolution Deep Dive

```
client.ResolveDependencies("bash")
  ↓
pkg/alpm/db.go:130 - Client.ResolveDependencies(packageName)
  
  visited := {}     # Packages fully processed
  visiting := {}    # Packages currently processing (cycle detection)
  result := []
  
  pkg ← Cache.GetPackage("bash") = {Name: bash, DependsOn: [glibc, ncurses]}
  
  resolveDepsRecursive(bash, visited, visiting, &result)
    ├─ Check: visiting[bash] → false (not circular)
    ├─ Check: visited[bash] → false (not done)
    ├─ visiting[bash] = true
    │
    ├─ For each dep in bash.DependsOn = [glibc, ncurses]:
    │  
    │  Dependency 1: "glibc>=2.0"
    │  ├─ ParseDependency("glibc>=2.0")
    │  │  └─ return (Dependency{Name: glibc, Constraint: >=2.0})
    │  │
    │  ├─ depPkg ← Cache.GetPackage("glibc", x86_64)
    │  │  = {Name: glibc, Version: 2.37-1, Repository: core}
    │  │
    │  ├─ Check constraint: CheckVersionConstraint("2.37-1", >=2.0)
    │  │  └─ VerCmp("2.37-1", "2.0") = 1 (greater) → OK
    │  │
    │  └─ resolveDepsRecursive(glibc, visited, visiting, &result)
    │     ├─ visiting[glibc] = true
    │     ├─ For each dep in glibc.DependsOn = [linux-api-headers, zlib]:
    │     │  
    │     │  Dependency 1: "linux-api-headers"
    │     │  ├─ depPkg ← Cache.GetPackage("linux-api-headers")
    │     │  ├─ resolveDepsRecursive(linux-api-headers, ...)
    │     │  │  ├─ visiting[linux-api-headers] = true
    │     │  │  ├─ No dependencies
    │     │  │  ├─ visited[linux-api-headers] = true
    │     │  │  └─ result.append(linux-api-headers)
    │     │  │     result = [linux-api-headers]
    │     │  │
    │     │  └─ (similar for zlib)
    │     │
    │     ├─ visited[glibc] = true
    │     ├─ result.append(glibc)
    │     └─ result = [linux-api-headers, zlib, glibc]
    │  
    │  Dependency 2: "ncurses"
    │  ├─ depPkg ← Cache.GetPackage("ncurses")
    │  ├─ resolveDepsRecursive(ncurses, ...)
    │  │  └─ result = [..., glibc, ncurses]
    │
    ├─ delete(visiting, bash)
    ├─ visited[bash] = true
    ├─ result.append(bash)
    └─ result = [linux-api-headers, zlib, glibc, ncurses, bash]

Return: [linux-api-headers, zlib, glibc, ncurses, bash]
  (dependencies before dependents)
```

---

## Version Comparison Algorithm

```
VerCmp("5.3.9-1", "5.3.10-1")
  ↓
pkg/alpm/version.go:12 - VerCmp(a, b)

  // Parse epochs
  epochA, releaseA ← splitEpoch("5.3.9-1")
    → epochA = "0", releaseA = "5.3.9-1"
  
  epochB, releaseB ← splitEpoch("5.3.10-1")
    → epochB = "0", releaseB = "5.3.10-1"
  
  // Compare epochs
  compareNumeric("0", "0") = 0 (equal, continue)
  
  // Parse release and revision
  relA, revA ← splitRevision("5.3.9-1")
    → relA = "5.3.9", revA = "1"
  
  relB, revB ← splitRevision("5.3.10-1")
    → relB = "5.3.10", revB = "1"
  
  // Compare releases
  compareRPMVersions("5.3.9", "5.3.10")
    ├─ tokenizeVersion("5.3.9")
    │  → [Seg(numeric, 5), Seg(non, .), Seg(numeric, 3), Seg(non, .), Seg(numeric, 9)]
    │
    ├─ tokenizeVersion("5.3.10")
    │  → [Seg(numeric, 5), Seg(non, .), Seg(numeric, 3), Seg(non, .), Seg(numeric, 10)]
    │
    └─ Compare segments:
       ├─ compareSegments(Seg(5), Seg(5)) = 0
       ├─ compareSegments(Seg(.), Seg(.)) = 0
       ├─ compareSegments(Seg(3), Seg(3)) = 0
       ├─ compareSegments(Seg(.), Seg(.)) = 0
       ├─ compareSegments(Seg(9), Seg(10))
       │  → numeric: 9 < 10 → return -1
       │
       └─ return -1 (a < b)
  
  // Compare revisions
  compareRPMVersions("1", "1") = 0
  
  return -1

Result: "5.3.9-1" < "5.3.10-1" ✓
```

---

## Package Database Parsing

```
client.RegisterSyncDB("core")
  ↓
pkg/alpm/db.go:24 - RegisterSyncDB(repo)
  
  ├─ LoadCachedDatabase("core")
  │  └─ pkg/alpm/parse.go:252
  │     ├─ dbPath ← /kod/db/sync/core.db
  │     ├─ data ← os.ReadFile(dbPath) [gzip compressed tar]
  │     │
  │     └─ parsePackageDatabase(data, arch)
  │        └─ pkg/alpm/parse.go:18
  │           ├─ Detect gzip magic bytes (0x1f 0x8b)
  │           ├─ gzip.NewReader() → decompress
  │           ├─ tar.NewReader()
  │           │
  │           ├─ For each tar entry:
  │           │  ├─ Read entry header
  │           │  ├─ Read entry content
  │           │  ├─ Parse path: "bash/desc", "bash/depends", etc.
  │           │  ├─ Store by package: currentPkg["bash:desc"] = content
  │           │
  │           ├─ Group files by package:
  │           │  pkgDirs["bash"] = {
  │           │    desc: "%NAME%\nbash\n%VERSION%\n5.3.9-1\n...",
  │           │    depends: "glibc\nncurses\n",
  │           │    optdepends: "",
  │           │    provides: "",
  │           │    conflicts: ""
  │           │  }
  │           │
  │           ├─ For each package group:
  │           │  ├─ parsePackageEntry(files, arch)
  │           │  │  └─ Extract metadata:
  │           │  │     ├─ NAME → bash
  │           │  │     ├─ VERSION → 5.3.9-1
  │           │  │     ├─ DESC → Bourne again shell
  │           │  │     ├─ ARCH → x86_64
  │           │  │     ├─ CSIZE → 654321
  │           │  │     ├─ ISIZE → 1234567
  │           │  │     ├─ depends → [glibc, ncurses]
  │           │  │     └─ ...
  │           │  │
  │           │  ├─ Filter by arch (x86_64, any only)
  │           │  ├─ Keep latest version if duplicates
  │           │  └─ packages[bash] = Package{...}
  │           │
  │           └─ return packages = {bash: Package, glibc: Package, ...}
  │
  ├─ Build Provides index:
  │  for each pkg in packages:
  │    for each virtual in pkg.Provides:
  │      provides[virtual] ← append pkg
  │
  └─ Return Database{
     Name: core,
     Path: /kod/db/sync/core.db,
     Packages: {bash: Package, ...},
     Provides: {virtual-name: [Package, ...]},
     Arch: x86_64
   }

Cache.AddDatabase(db)
  ├─ Merge all packages into cache
  ├─ Respect repo precedence (core > extra > community)
  ├─ For duplicate packages: keep one from highest-priority repo
  └─ Merge provides mappings
```

---

## Download Pipeline

```
downloader.DownloadPackages([bash, vim, git])
  ↓
pkg/download/download.go:103

  semaphore ← make(chan, maxConcurrent=5)
  results ← {}
  
  For each package (concurrent via goroutine):
    ├─ Acquire semaphore: semaphore ← struct{}{}
    │
    ├─ DownloadPackage(bash)
    │  └─ pkg/download/download.go:45
    │     ├─ os.MkdirAll(/kod/cache, 0755)
    │     │
    │     ├─ filename ← bash-5.3.9-1-x86_64.pkg.tar.zst
    │     ├─ pkgURL ← https://mirror.rackspace.com/archlinux/core/os/x86_64/bash-5.3.9-1-x86_64.pkg.tar.zst
    │     │
    │     ├─ resp ← httpClient.Get(pkgURL)
    │     ├─ if resp.StatusCode != 200: error
    │     │
    │     ├─ tmpPath ← /kod/cache/bash-5.3.9-1-x86_64.pkg.tar.zst.tmp
    │     ├─ tmpFile ← os.Create(tmpPath)
    │     ├─ written ← io.Copy(tmpFile, resp.Body)
    │     ├─ tmpFile.Close()
    │     │
    │     ├─ finalPath ← /kod/cache/bash-5.3.9-1-x86_64.pkg.tar.zst
    │     ├─ os.Rename(tmpPath, finalPath) [atomic]
    │     └─ return finalPath
    │
    ├─ results[bash] ← /kod/cache/bash-5.3.9-1-x86_64.pkg.tar.zst
    │
    └─ Release semaphore: <-semaphore

  WaitGroup.Wait() [wait for all goroutines]
  
  return results = {
    bash: /kod/cache/bash-5.3.9-1-x86_64.pkg.tar.zst,
    vim: /kod/cache/vim-9.0.123-1-x86_64.pkg.tar.zst,
    git: /kod/cache/git-2.38.1-1-x86_64.pkg.tar.zst
  }
```

---

## Extraction Pipeline

```
storeManager.ExtractPackage(/kod/cache/bash-5.3.9-1-x86_64.pkg.tar.zst, bash, 5.3.9-1)
  ↓
pkg/store/store.go:50

  destDir ← /kod/store/bash/5.3.9-1
  
  extractor.ExtractPackage(pkgPath, destDir)
    └─ pkg/extract/extract.go:39
       ├─ os.Open(pkgPath)
       ├─ zstd.NewReader(file) [decompress]
       ├─ tar.NewReader(decompressed)
       │
       ├─ os.MkdirAll(destDir, 0755)
       │
       └─ For each tar entry:
          ├─ header ← tarReader.Next()
          ├─ if header.Name == "usr/bin/bash":
          │  ├─ parentDir ← /kod/store/bash/5.3.9-1/usr/bin
          │  ├─ os.MkdirAll(parentDir, 0755)
          │  ├─ targetPath ← /kod/store/bash/5.3.9-1/usr/bin/bash
          │  ├─ file ← os.Create(targetPath)
          │  ├─ io.Copy(file, tarReader) [extract content]
          │  ├─ file.Close()
          │  ├─ os.Chmod(targetPath, 0755) [preserve perms]
          │  └─ extractedFiles.append({
          │       Path: usr/bin/bash,
          │       AbsPath: /kod/store/bash/5.3.9-1/usr/bin/bash,
          │       Size: 123456,
          │       Mode: 0755
          │     })
          │
          ├─ else if header.Typeflag == TypeDir:
          │  ├─ os.MkdirAll(targetPath, 0755)
          │  └─ extractedFiles.append({IsDirectory: true, ...})
          │
          └─ else if header.Typeflag == TypeSymlink:
             ├─ os.Symlink(header.Linkname, targetPath)
             └─ extractedFiles.append({IsSymlink: true, LinkTarget: ..., ...})
       
       return []ExtractedFile{...}

  return extractedFiles
```

---

## Registry Update Flow

```
reg := registry.NewRegistry(/kod/registry.json)
  ↓
pkg/registry/registry.go:35 - NewRegistry(path)

  ├─ Load existing registry
  │  └─ os.ReadFile(/kod/registry.json)
  │  └─ json.Unmarshal() → map[string]*Package
  │
  └─ return Registry{
       path: /kod/registry.json,
       packages: {
         "curl": {name: curl, version: 7.x.x, ...},
         "bash": {name: bash, version: 5.2.x, ...}
       }
     }

For each installed package:
  
  regPkg := &registry.Package{
    Name: "bash",
    Version: "5.3.9-1",
    Files: [usr/bin/bash, usr/share/doc/bash/README, ...],
    Executables: [usr/bin/bash],
    Dependencies: [glibc, ncurses, readline],
    InstallDate: "2024-01-15T14:30:00Z"
  }
  
  reg.AddPackage(regPkg)
    └─ r.packages[bash] = regPkg
  
  reg.Save()
    ├─ json.MarshalIndent(r.packages, "", "  ")
    └─ os.WriteFile(/kod/registry.json, data, 0644)
       
       Resulting JSON:
       {
         "bash": {
           "name": "bash",
           "version": "5.3.9-1",
           "files": [...],
           "executables": ["usr/bin/bash"],
           "dependencies": ["glibc", "ncurses", "readline"],
           "install_date": "2024-01-15T14:30:00Z"
         }
       }
```

---

## Configuration Loading Flow

```
main() → handleInstall() → loadConfig()
  ↓
cmd/chisel/main.go:152 - loadConfig()

  cfgPath := ""
  
  // Priority 1: Command-line flag
  if configPath != "" (from flag.Parse):
    cfgPath = configPath
  
  // Priority 2: Environment variable
  else if env CHISEL_CONFIG != "":
    cfgPath = env
  
  // Priority 3: Default
  else:
    cfgPath = /etc/chisel/config.json
  
  cfg, err := config.Load(cfgPath)
    └─ pkg/config/config.go:104
       ├─ os.ReadFile(cfgPath)
       ├─ json.Unmarshal(data) → Config struct
       ├─ cfg.Normalize() [fill defaults for empty fields]
       └─ return cfg
  
  if err != nil:
    ├─ Print warning
    └─ cfg = config.DefaultConfig()
       └─ pkg/config/config.go:80
          return Config{
            BaseDir: /kod,
            StoreRoot: /kod/store,
            RegistryPath: /kod/registry.json,
            DBPath: /kod/db/sync,
            CachePath: /kod/cache,
            WrapperDir: /kod/wrappers,
            MirrorURL: https://mirror.rackspace.com/archlinux,
            Architecture: x86_64,
            Repositories: [core, extra, community],
            MaxConcurrentDownloads: 5,
            DownloadTimeout: 300,
            KeepVersions: 3
          }
  
  // Apply command-line overrides
  if baseDir != "" (from --base-dir flag):
    cfg.BaseDir = baseDir
    cfg.UpdateDerivedPaths()
  
  if mirrorURL != "" (from --mirror flag):
    cfg.MirrorURL = mirrorURL
  
  return cfg
```

---

## Error Handling Flows

### Circular Dependency Detection

```
resolveDepsRecursive(pkg-a, visited, visiting, &result)
  ├─ visiting[pkg-a] = true
  ├─ For each dep in pkg-a.DependsOn:
  │  ├─ Find dep-pkg
  │  └─ resolveDepsRecursive(dep-pkg, ...)
  │     ├─ visiting[dep-pkg] = true
  │     ├─ For each sub-dep in dep-pkg.DependsOn:
  │     │  └─ resolveDepsRecursive(sub-pkg-a, ...)
  │     │     ├─ if visiting[pkg-a] == true: CYCLE DETECTED!
  │     │     ├─ return ResolutionError{
  │     │     │    Reason: "circular dependency at pkg-a",
  │     │     │    Cycle: [pkg-a]
  │     │     │  }

Error returned up stack → InstallCommand.Run() → main() → os.Exit(1)
```

### Missing Dependency

```
resolveDepsRecursive(...):
  ├─ depPkg ← Cache.GetPackage(depName)
  ├─ if depPkg == nil:
  │  └─ providers ← Cache.GetProvidingPackages(depName)
  │     ├─ if len(providers) == 0:
  │     │  └─ return ResolutionError{
  │     │       Reason: "dependency X not found (required by Y)"
  │     │     }
```

### Download Failure (Partial)

```
downloader.DownloadPackages([...]):
  
  For each package:
    ├─ path, err ← DownloadPackage(pkg)
    ├─ if err != nil:
    │  └─ downloadErrors.append(err)
  
  if len(downloadErrors) > 0:
    └─ collect all errors
    └─ return (partial results, combined error)

InstallCommand.Run():
  ├─ if err != nil:
  │  ├─ Print warning: "Download warnings/errors: ..."
  │  ├─ Continue with successfully downloaded packages
  │  └─ (or abort if critical)
```

### Path Traversal Attack Prevention

```
ExtractPackage():
  ├─ destDir ← /kod/store/bash/5.3.9-1
  │
  ├─ For each tar entry:
  │  ├─ targetPath ← filepath.Join(destDir, header.Name)
  │  ├─ if !filepath.HasPrefix(clean(targetPath), clean(destDir)):
  │  │  └─ ERROR: archive contains path outside destination
  │  │
  │  Example attack prevented:
  │  ├─ destDir = /kod/store/bash/5.3.9-1
  │  ├─ header.Name = ../../../../etc/passwd
  │  ├─ targetPath = /kod/store/bash/5.3.9-1/../../../../etc/passwd
  │  ├─ clean(targetPath) = /etc/passwd
  │  ├─ HasPrefix(/etc/passwd, /kod/store/bash/5.3.9-1) = false
  │  └─ Return error ✓
```

