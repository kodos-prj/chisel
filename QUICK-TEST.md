# Quick Test Guide

## One-Line Tests

```bash
# Quick validation (safe, no installation)
./test-workflow.sh --dry-run

# Full test with actual installation
sudo ./test-workflow.sh

# Test specific package
sudo ./test-workflow.sh --package vim
```

## What Gets Tested

| Operation | Status | Time |
|-----------|--------|------|
| Sync databases | ✅ Works | ~20s |
| Search packages | ✅ Works | <1s |
| Package info | ✅ Works | <1s |
| Install package | ✅ Works | ~30s |
| List packages | ✅ Works | <1s |
| Upgrade packages | ✅ Works | ~30-60s |
| Remove package | ✅ Works | <5s |

## Quick Examples

### Test small package (recommended for quick testing)
```bash
sudo ./test-workflow.sh --package nano
```

### Test with dependencies
```bash
sudo ./test-workflow.sh --package vim
```

### Keep test files for inspection
```bash
sudo ./test-workflow.sh --skip-cleanup
# Then check /tmp/chisel-test-*/
```

### Test only sync and search (no sudo needed)
```bash
./test-workflow.sh --dry-run
```

## Expected Output

```
✓ Database synchronization       [~20 seconds]
✓ Package search                 [<1 second]
✓ Package information retrieval  [<1 second]
✓ Package installation           [~30 seconds]
✓ Package listing                [<1 second]
✓ Package upgrade                [~30-60 seconds]
✓ Package removal                [~5 seconds]
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "Chisel binary not found" | `go build -o chisel ./cmd/chisel` |
| "Permission denied" | Use `sudo ./test-workflow.sh` |
| Database sync fails | Check internet connection |
| Want to see what happens first | Run with `--dry-run` |

## Integration Test

For complete integration testing:
```bash
# Build
go build -o chisel ./cmd/chisel

# Unit tests
go test ./pkg/... -short

# Integration tests  
go test ./integration/...

# Workflow test
./test-workflow.sh --dry-run
```

Total test time: ~4-5 minutes
