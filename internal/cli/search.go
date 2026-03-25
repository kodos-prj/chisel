// Package cli implements command-line interface commands for packmgr.
package cli

import (
	"fmt"

	"github.com/yourusername/packmgr-go/pkg/alpm"
	"github.com/yourusername/packmgr-go/pkg/config"
)

// SearchCommand implements the 'packmgr search' command.
// It searches for packages in the synced databases.
type SearchCommand struct {
	config *config.Config
}

// NewSearchCommand creates a new search command instance.
func NewSearchCommand(cfg *config.Config) *SearchCommand {
	return &SearchCommand{
		config: cfg,
	}
}

// Execute runs the search command.
// pattern is the package name or pattern to search for.
func (s *SearchCommand) Execute(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("search pattern cannot be empty")
	}

	// Initialize ALPM client
	// Note: We pass AlpmDBPath (the parent directory), not DBPath (the sync directory)
	client, err := alpm.NewClient(s.config.AlpmRoot, s.config.AlpmDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize ALPM: %w", err)
	}
	defer client.Close()

	// Register sync databases
	if err := client.RegisterAllSyncDBs(s.config.Repositories); err != nil {
		return fmt.Errorf("failed to register sync databases: %w", err)
	}

	// Search for packages
	packages, err := client.SearchPackages(pattern)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Display results
	fmt.Printf("Found %d package(s) matching '%s':\n\n", len(packages), pattern)

	for _, pkg := range packages {
		fmt.Printf("%s/%s %s\n", pkg.DB().Name(), pkg.Name(), pkg.Version())
		if desc := pkg.Description(); desc != "" {
			fmt.Printf("    %s\n", desc)
		}
		fmt.Println()
	}

	return nil
}

// ExactSearch searches for an exact package name.
func (s *SearchCommand) ExactSearch(name string) error {
	if name == "" {
		return fmt.Errorf("package name cannot be empty")
	}

	// Initialize ALPM client
	// Note: We pass AlpmDBPath (the parent directory), not DBPath (the sync directory)
	client, err := alpm.NewClient(s.config.AlpmRoot, s.config.AlpmDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize ALPM: %w", err)
	}
	defer client.Close()

	// Register sync databases
	if err := client.RegisterAllSyncDBs(s.config.Repositories); err != nil {
		return fmt.Errorf("failed to register sync databases: %w", err)
	}

	// Search for exact package
	pkg, err := client.SearchPackage(name)
	if err != nil {
		return fmt.Errorf("package not found: %w", err)
	}

	// Display result
	fmt.Printf("%s/%s %s\n", pkg.DB().Name(), pkg.Name(), pkg.Version())
	if desc := pkg.Description(); desc != "" {
		fmt.Printf("    %s\n", desc)
	}

	return nil
}
