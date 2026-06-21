package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kodos-prj/chisel/pkg/config"
	"github.com/kodos-prj/chisel/pkg/registry"
)

// InstallScriptsCommand executes post_install/post_upgrade scripts for packages
type InstallScriptsCommand struct {
	config *config.Config
}

// NewInstallScriptsCommand creates a new install-scripts command
func NewInstallScriptsCommand(cfg *config.Config) *InstallScriptsCommand {
	return &InstallScriptsCommand{
		config: cfg,
	}
}

// Execute runs install scripts for specified packages (or all if none specified)
// chrootDir: empty string for non-chroot mode, path for chroot mode
func (i *InstallScriptsCommand) Execute(packageNames []string, verbose bool, chrootDir string) error {
	// Load current registry
	reg, err := registry.NewRegistry(i.config.RegistryPath)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Filter packages with install scripts
	var packagesToRun []*registry.Package
	if len(packageNames) > 0 {
		// Filter by specified names
		for _, name := range packageNames {
			pkg, ok := reg.GetPackage(name)
			if !ok {
				fmt.Fprintf(os.Stderr, "⚠ Warning: Package %s not found in registry\n", name)
				continue
			}
			if pkg.HasInstallScript {
				packagesToRun = append(packagesToRun, pkg)
			} else {
				fmt.Printf("ℹ %s: No install script\n", name)
			}
		}
	} else {
		// All packages with install scripts
		for _, pkg := range reg.ListPackages() {
			if pkg.HasInstallScript {
				packagesToRun = append(packagesToRun, pkg)
			}
		}
	}

	if len(packagesToRun) == 0 {
		fmt.Println("No install scripts to run")
		return nil
	}

	// Determine execution mode
	mode := "current system context"
	if chrootDir != "" {
		mode = fmt.Sprintf("chroot %s", chrootDir)
	}
	fmt.Printf("Running install scripts (%s) for %d package(s)...\n", mode, len(packagesToRun))

	successCount := 0
	failureCount := 0

	for _, pkg := range packagesToRun {
		// Determine operation: post_install (new) or post_upgrade (upgraded)
		operation := "post_install"
		// For now, we'll default to post_install if re-running via install-scripts
		// The package already exists in registry, but we check if this is the original install or an update
		// Since we can't track execution status, we default to post_install
		// The user should understand that multiple runs of the same script should be idempotent
		if verbose {
			fmt.Printf("Running %s for %s/%s...\n", operation, pkg.Name, pkg.Version)
		}

		if err := i.runInstallScript(pkg, operation, chrootDir); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: Install script failed (%s): %v\n", pkg.Name, operation, err)
			failureCount++
			// Continue with next package
			continue
		}

		successCount++
		fmt.Printf("  ✓ %s: %s completed\n", pkg.Name, operation)
	}

	fmt.Printf("\n✓ %d install script(s) executed", successCount)
	if failureCount > 0 {
		fmt.Printf(", %d failed\n", failureCount)
	} else {
		fmt.Println()
	}

	return nil
}

// runInstallScript dispatches to the appropriate execution method based on chrootDir
func (i *InstallScriptsCommand) runInstallScript(pkg *registry.Package, operation string, chrootDir string) error {
	if chrootDir == "" {
		return i.runInstallScriptDirect(pkg, operation)
	}
	return i.runInstallScriptChroot(pkg, operation, chrootDir)
}

// runInstallScriptDirect executes an install script directly in the current system context
func (i *InstallScriptsCommand) runInstallScriptDirect(pkg *registry.Package, operation string) error {
	// Path: /kod/store/<name>/<version>/.INSTALL
	extractDir := filepath.Join(i.config.StoreRoot, pkg.Name, pkg.Version)
	scriptPath := filepath.Join(extractDir, ".INSTALL")

	// Verify script exists
	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("script not found at %s", scriptPath)
	}

	// Execute script from root directory context (/), allowing relative paths in scripts to resolve correctly
	// Source the .INSTALL file using absolute path, then call the function
	shellCmd := fmt.Sprintf("source '%s' && %s", scriptPath, operation)
	cmd := exec.Command("bash", "-c", shellCmd)

	// Capture output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// runInstallScriptChroot executes an install script in chroot context
func (i *InstallScriptsCommand) runInstallScriptChroot(pkg *registry.Package, operation string, chrootDir string) error {
	// Execute script from root directory context, allowing relative paths in scripts to resolve correctly
	// In chroot, /kod/store/<name>/<version>/.INSTALL is the correct absolute path
	scriptPath := filepath.Join("/kod/store", pkg.Name, pkg.Version, ".INSTALL")
	shellCmd := fmt.Sprintf("source %s && %s", scriptPath, operation)

	cmd := exec.Command("chroot", chrootDir, "bash", "-c", shellCmd)

	// Capture output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
