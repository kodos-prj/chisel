package alpm

import (
	"fmt"
	"os"
	"os/exec"
)

// VerifyDatabaseSignature verifies the GPG signature of a database file.
// It uses the system's gpg binary to verify the signature.
// Parameters:
//   - dbPath: path to the database file
//   - sigPath: path to the .sig signature file
//
// Returns error if verification fails or gpg is not available.
func VerifyDatabaseSignature(dbPath, sigPath string) error {
	// Check if gpg is available
	gpgPath, err := exec.LookPath("gpg")
	if err != nil {
		return fmt.Errorf("gpg not found in PATH, skipping signature verification: %w", err)
	}

	// Check if signature file exists
	if _, err := os.Stat(sigPath); err != nil {
		return fmt.Errorf("signature file not found: %w", err)
	}

	// Run: gpg --verify sigPath dbPath
	cmd := exec.Command(gpgPath, "--verify", sigPath, dbPath)

	// Capture output for error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("GPG signature verification failed: %s\n%s", err, string(output))
	}

	return nil
}

// VerifyDatabaseSignatureOptional attempts to verify a database signature.
// Unlike VerifyDatabaseSignature, this returns a warning instead of error if gpg is not available.
// Returns (verified, error) where verified is true if signature is valid or gpg is not available.
func VerifyDatabaseSignatureOptional(dbPath, sigPath string) (bool, error) {
	// Check if gpg is available
	gpgPath, err := exec.LookPath("gpg")
	if err != nil {
		// gpg not available, return success with warning
		return true, fmt.Errorf("gpg not found in PATH, database signature verification skipped")
	}

	// Check if signature file exists
	if _, err := os.Stat(sigPath); err != nil {
		// No signature file, return success
		return true, nil
	}

	// Run: gpg --verify sigPath dbPath
	cmd := exec.Command(gpgPath, "--verify", sigPath, dbPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("GPG signature verification failed: %s\n%s", err, string(output))
	}

	return true, nil
}

// CheckGPGAvailable checks if gpg is available on the system.
func CheckGPGAvailable() bool {
	_, err := exec.LookPath("gpg")
	return err == nil
}

// VerifyDatabaseIntegrity verifies the integrity of a database file.
// This checks if the database file is a valid tar.gz and can be read.
// Returns error if the database is corrupted.
func VerifyDatabaseIntegrity(dbPath string) error {
	data, err := os.ReadFile(dbPath)
	if err != nil {
		return fmt.Errorf("failed to read database file: %w", err)
	}

	// Try to decompress and read tar structure
	_, err = parsePackageDatabase(data, "x86_64") // Use x86_64 as default for verification
	if err != nil {
		return fmt.Errorf("database integrity check failed: %w", err)
	}

	return nil
}
