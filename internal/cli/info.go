// Package cli implements command-line interface commands for packmgr.
package cli

import (
	"fmt"

	"github.com/yourusername/packmgr-go/pkg/alpm"
	"github.com/yourusername/packmgr-go/pkg/config"
)

// InfoCommand implements the 'packmgr info' command.
// It displays detailed information about a package.
type InfoCommand struct {
	config *config.Config
}

// NewInfoCommand creates a new info command instance.
func NewInfoCommand(cfg *config.Config) *InfoCommand {
	return &InfoCommand{
		config: cfg,
	}
}

// Execute runs the info command.
// name is the package name to get information about.
func (i *InfoCommand) Execute(name string) error {
	if name == "" {
		return fmt.Errorf("package name cannot be empty")
	}

	// Initialize ALPM client
	// Note: We pass AlpmDBPath (the parent directory), not DBPath (the sync directory)
	client, err := alpm.NewClient(i.config.AlpmRoot, i.config.AlpmDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize ALPM: %w", err)
	}
	defer client.Close()

	// Register sync databases
	if err := client.RegisterAllSyncDBs(i.config.Repositories); err != nil {
		return fmt.Errorf("failed to register sync databases: %w", err)
	}

	// Get package info
	info, err := client.GetPackageInfo(name)
	if err != nil {
		return fmt.Errorf("failed to get package info: %w", err)
	}

	// Display detailed information
	fmt.Printf("Repository      : %s\n", info.Repository)
	fmt.Printf("Name            : %s\n", info.Name)
	fmt.Printf("Version         : %s\n", info.Version)
	fmt.Printf("Description     : %s\n", info.Description)
	fmt.Printf("Architecture    : %s\n", info.Architecture)
	fmt.Printf("URL             : %s\n", info.URL)

	if len(info.Licenses) > 0 {
		fmt.Printf("Licenses        : %v\n", info.Licenses)
	}

	if len(info.Groups) > 0 {
		fmt.Printf("Groups          : %v\n", info.Groups)
	}

	if len(info.Provides) > 0 {
		fmt.Printf("Provides        : %v\n", info.Provides)
	}

	if len(info.DependsOn) > 0 {
		fmt.Printf("Depends On      : %v\n", info.DependsOn)
	}

	if len(info.OptDepends) > 0 {
		fmt.Printf("Optional Deps   : %v\n", info.OptDepends)
	}

	if len(info.Conflicts) > 0 {
		fmt.Printf("Conflicts With  : %v\n", info.Conflicts)
	}

	if len(info.Replaces) > 0 {
		fmt.Printf("Replaces        : %v\n", info.Replaces)
	}

	fmt.Printf("Download Size   : %.2f MB\n", float64(info.DownloadSize)/(1024*1024))
	fmt.Printf("Installed Size  : %.2f MB\n", float64(info.Size)/(1024*1024))
	fmt.Printf("Packager        : %s\n", info.Maintainer)

	if info.PackageBase != "" {
		fmt.Printf("Package Base    : %s\n", info.PackageBase)
	}

	return nil
}

// ExecuteWithDeps runs the info command and also shows dependency tree.
func (i *InfoCommand) ExecuteWithDeps(name string) error {
	// First show basic info
	if err := i.Execute(name); err != nil {
		return err
	}

	// Initialize ALPM client for dependency resolution
	// Note: We pass AlpmDBPath (the parent directory), not DBPath (the sync directory)
	client, err := alpm.NewClient(i.config.AlpmRoot, i.config.AlpmDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize ALPM: %w", err)
	}
	defer client.Close()

	// Register sync databases
	if err := client.RegisterAllSyncDBs(i.config.Repositories); err != nil {
		return fmt.Errorf("failed to register sync databases: %w", err)
	}

	// Resolve dependencies
	deps, err := client.ResolveDependencies(name)
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	// Display dependency tree
	fmt.Println("\nComplete Dependency Tree (install order):")
	for idx, dep := range deps {
		fmt.Printf("  %d. %s\n", idx+1, dep)
	}

	return nil
}
