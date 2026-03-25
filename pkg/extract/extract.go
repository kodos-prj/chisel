// Package extract handles extraction of .pkg.tar.zst archives.
package extract

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/klauspost/compress/zstd"
)

// Extractor handles extraction of Arch Linux package archives.
type Extractor struct {
	preservePerms bool // Preserve original file permissions
}

// NewExtractor creates a new package extractor.
func NewExtractor(preservePerms bool) *Extractor {
	return &Extractor{
		preservePerms: preservePerms,
	}
}

// ExtractedFile represents a file that was extracted from an archive.
type ExtractedFile struct {
	Path        string // Relative path in the archive
	AbsPath     string // Absolute path where extracted
	IsDirectory bool
	IsSymlink   bool   // True if this is a symbolic link
	LinkTarget  string // Target of the symlink (if IsSymlink == true)
	Size        int64
	Mode        os.FileMode
}

// ExtractPackage extracts a .pkg.tar.zst file to the destination directory.
// Returns a list of extracted files and any error.
func (e *Extractor) ExtractPackage(pkgPath, destDir string) ([]ExtractedFile, error) {
	// Open the package file
	pkgFile, err := os.Open(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open package file: %w", err)
	}
	defer pkgFile.Close()

	// Create zstd decoder
	decoder, err := zstd.NewReader(pkgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
	}
	defer decoder.Close()

	// Create tar reader
	tarReader := tar.NewReader(decoder)

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	var extractedFiles []ExtractedFile

	// Extract each file from the archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return extractedFiles, fmt.Errorf("failed to read archive: %w", err)
		}

		// Handle all file types: regular files, directories, symlinks, and hard links
		switch header.Typeflag {
		case tar.TypeReg, tar.TypeDir, tar.TypeSymlink, tar.TypeLink:
			// These are the types we want to extract
		default:
			// Skip other types (device files, FIFOs, etc.)
			continue
		}

		// Build destination path
		// Handle both normal paths and .MTREE/.PKGINFO special files
		targetPath := filepath.Join(destDir, header.Name)

		// Prevent directory traversal attacks
		if !filepath.HasPrefix(filepath.Clean(targetPath), filepath.Clean(destDir)) {
			return extractedFiles, fmt.Errorf("archive contains path outside destination: %s", header.Name)
		}

		if header.Typeflag == tar.TypeDir {
			// Create directory
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return extractedFiles, fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}

			extractedFiles = append(extractedFiles, ExtractedFile{
				Path:        header.Name,
				AbsPath:     targetPath,
				IsDirectory: true,
				Mode:        os.FileMode(header.Mode),
			})
		} else if header.Typeflag == tar.TypeSymlink {
			// Create symlink
			parentDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return extractedFiles, fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Remove existing symlink if it exists
			_ = os.Remove(targetPath)

			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return extractedFiles, fmt.Errorf("failed to create symlink %s -> %s: %w", targetPath, header.Linkname, err)
			}

			extractedFiles = append(extractedFiles, ExtractedFile{
				Path:       header.Name,
				AbsPath:    targetPath,
				IsSymlink:  true,
				LinkTarget: header.Linkname,
				Mode:       os.FileMode(header.Mode),
			})
		} else if header.Typeflag == tar.TypeLink {
			// Create hard link
			parentDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return extractedFiles, fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Hard link target path is also relative to destDir
			linkTargetPath := filepath.Join(destDir, header.Linkname)

			// Remove existing file if it exists
			_ = os.Remove(targetPath)

			if err := os.Link(linkTargetPath, targetPath); err != nil {
				return extractedFiles, fmt.Errorf("failed to create hard link %s -> %s: %w", targetPath, linkTargetPath, err)
			}

			extractedFiles = append(extractedFiles, ExtractedFile{
				Path:       header.Name,
				AbsPath:    targetPath,
				IsSymlink:  true, // Mark as symlink for our purposes
				LinkTarget: header.Linkname,
				Mode:       os.FileMode(header.Mode),
			})
		} else {
			// Regular file (tar.TypeReg)
			// Create parent directory if needed
			parentDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return extractedFiles, fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Create file
			file, err := os.Create(targetPath)
			if err != nil {
				return extractedFiles, fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			// Copy file contents
			written, err := io.Copy(file, tarReader)
			file.Close()
			if err != nil {
				os.Remove(targetPath)
				return extractedFiles, fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}

			// Set file permissions if preserving
			if e.preservePerms {
				if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
					// Log but don't fail on permission issues
					fmt.Fprintf(os.Stderr, "warning: failed to set permissions on %s: %v\n", targetPath, err)
				}
			}

			extractedFiles = append(extractedFiles, ExtractedFile{
				Path:    header.Name,
				AbsPath: targetPath,
				Size:    written,
				Mode:    os.FileMode(header.Mode),
			})
		}
	}

	return extractedFiles, nil
}

// ExtractFile extracts a single file from a package archive.
func (e *Extractor) ExtractFile(pkgPath, fileName, destDir string) error {
	pkgFile, err := os.Open(pkgPath)
	if err != nil {
		return fmt.Errorf("failed to open package file: %w", err)
	}
	defer pkgFile.Close()

	decoder, err := zstd.NewReader(pkgFile)
	if err != nil {
		return fmt.Errorf("failed to create zstd decoder: %w", err)
	}
	defer decoder.Close()

	tarReader := tar.NewReader(decoder)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return fmt.Errorf("file %s not found in archive", fileName)
		}
		if err != nil {
			return fmt.Errorf("failed to read archive: %w", err)
		}

		if header.Name == fileName {
			if header.Typeflag == tar.TypeDir {
				return os.MkdirAll(filepath.Join(destDir, fileName), 0755)
			}

			// Create destination path
			targetPath := filepath.Join(destDir, fileName)
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			file, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer file.Close()

			if _, err := io.Copy(file, tarReader); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}

			return nil
		}
	}
}

// ListContents lists all files in a package archive without extracting.
func (e *Extractor) ListContents(pkgPath string) ([]string, error) {
	pkgFile, err := os.Open(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open package file: %w", err)
	}
	defer pkgFile.Close()

	decoder, err := zstd.NewReader(pkgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
	}
	defer decoder.Close()

	tarReader := tar.NewReader(decoder)

	var files []string
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return files, fmt.Errorf("failed to read archive: %w", err)
		}

		files = append(files, header.Name)
	}

	return files, nil
}
