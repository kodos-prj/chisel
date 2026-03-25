// Package database handles synchronization of Arch Linux package databases.
package database

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Syncer handles downloading and managing Arch Linux package databases.
type Syncer struct {
	mirrorURL  string
	dbPath     string
	arch       string
	httpClient *http.Client
}

// NewSyncer creates a new database syncer.
func NewSyncer(mirrorURL, dbPath, arch string, timeout time.Duration) *Syncer {
	return &Syncer{
		mirrorURL: mirrorURL,
		dbPath:    dbPath,
		arch:      arch,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Sync downloads the specified repositories' databases from the Arch mirror.
func (s *Syncer) Sync(repos []string) error {
	// Ensure database directory exists
	if err := os.MkdirAll(s.dbPath, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Download each repository database
	for _, repo := range repos {
		if err := s.downloadDatabase(repo); err != nil {
			return fmt.Errorf("failed to sync %s: %w", repo, err)
		}
	}

	return nil
}

// downloadDatabase downloads a single repository database.
func (s *Syncer) downloadDatabase(repo string) error {
	// Construct database URL
	// Format: https://mirror.rackspace.com/archlinux/core/os/x86_64/core.db
	dbURL := fmt.Sprintf("%s/%s/os/%s/%s.db", s.mirrorURL, repo, s.arch, repo)

	fmt.Printf("Downloading %s database from %s...\n", repo, dbURL)

	// Download the database
	resp, err := s.httpClient.Get(dbURL)
	if err != nil {
		return fmt.Errorf("failed to download database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("database download failed with status: %s", resp.Status)
	}

	// Create temporary file
	tmpFile := filepath.Join(s.dbPath, fmt.Sprintf("%s.db.tmp", repo))
	f, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer f.Close()

	// Copy downloaded data to file
	written, err := io.Copy(f, resp.Body)
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to write database: %w", err)
	}

	// Close file before rename
	f.Close()

	// Atomic rename to final location
	finalPath := filepath.Join(s.dbPath, fmt.Sprintf("%s.db", repo))
	if err := os.Rename(tmpFile, finalPath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to move database to final location: %w", err)
	}

	fmt.Printf("✓ Downloaded %s.db (%d bytes)\n", repo, written)
	return nil
}

// LastSyncTime returns the last modification time of a repository database.
func (s *Syncer) LastSyncTime(repo string) (time.Time, error) {
	dbFile := filepath.Join(s.dbPath, fmt.Sprintf("%s.db", repo))
	info, err := os.Stat(dbFile)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil // Never synced
		}
		return time.Time{}, fmt.Errorf("failed to stat database: %w", err)
	}
	return info.ModTime(), nil
}

// DatabaseExists checks if a repository database has been downloaded.
func (s *Syncer) DatabaseExists(repo string) bool {
	dbFile := filepath.Join(s.dbPath, fmt.Sprintf("%s.db", repo))
	_, err := os.Stat(dbFile)
	return err == nil
}
