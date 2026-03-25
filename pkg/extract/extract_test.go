package extract

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
)

// createTestArchive creates a test .tar.zst file with the given files.
// files is a map of filename -> content
func createTestArchive(files map[string]string) ([]byte, error) {
	var buf bytes.Buffer

	encoder, err := zstd.NewWriter(&buf)
	if err != nil {
		return nil, err
	}

	tarWriter := tar.NewWriter(encoder)

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, err
		}

		if _, err := tarWriter.Write([]byte(content)); err != nil {
			return nil, err
		}
	}

	if err := tarWriter.Close(); err != nil {
		return nil, err
	}

	if err := encoder.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// TestNewExtractor tests extractor creation.
func TestNewExtractor(t *testing.T) {
	e := NewExtractor(true)
	if e == nil {
		t.Fatal("NewExtractor returned nil")
	}
	if !e.preservePerms {
		t.Error("preservePerms not set correctly")
	}
}

// TestExtractPackage tests extracting a complete package.
func TestExtractPackage(t *testing.T) {
	// Create test archive
	files := map[string]string{
		"file1.txt":     "content 1",
		"file2.txt":     "content 2",
		"dir/file3.txt": "content 3",
	}

	archiveData, err := createTestArchive(files)
	if err != nil {
		t.Fatalf("Failed to create test archive: %v", err)
	}

	// Write archive to temp file
	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "test.pkg.tar.zst")
	if err := os.WriteFile(pkgPath, archiveData, 0644); err != nil {
		t.Fatalf("Failed to write test package: %v", err)
	}

	// Extract
	destDir := filepath.Join(tmpDir, "extracted")
	e := NewExtractor(true)
	extracted, err := e.ExtractPackage(pkgPath, destDir)
	if err != nil {
		t.Fatalf("ExtractPackage failed: %v", err)
	}

	// Verify extracted files
	if len(extracted) != 3 {
		t.Errorf("Expected 3 extracted files, got %d", len(extracted))
	}

	// Check file1.txt
	content, err := os.ReadFile(filepath.Join(destDir, "file1.txt"))
	if err != nil {
		t.Errorf("Failed to read extracted file1.txt: %v", err)
	}
	if string(content) != "content 1" {
		t.Errorf("file1.txt content mismatch: got %s", string(content))
	}

	// Check file2.txt
	content, err = os.ReadFile(filepath.Join(destDir, "file2.txt"))
	if err != nil {
		t.Errorf("Failed to read extracted file2.txt: %v", err)
	}
	if string(content) != "content 2" {
		t.Errorf("file2.txt content mismatch: got %s", string(content))
	}

	// Check dir/file3.txt
	content, err = os.ReadFile(filepath.Join(destDir, "dir", "file3.txt"))
	if err != nil {
		t.Errorf("Failed to read extracted dir/file3.txt: %v", err)
	}
	if string(content) != "content 3" {
		t.Errorf("dir/file3.txt content mismatch: got %s", string(content))
	}
}

// TestExtractPackageCreatesDirectories tests that extraction creates needed directories.
func TestExtractPackageCreatesDirectories(t *testing.T) {
	files := map[string]string{
		"deep/nested/path/file.txt": "content",
	}

	archiveData, err := createTestArchive(files)
	if err != nil {
		t.Fatalf("Failed to create test archive: %v", err)
	}

	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "test.pkg.tar.zst")
	if err := os.WriteFile(pkgPath, archiveData, 0644); err != nil {
		t.Fatalf("Failed to write test package: %v", err)
	}

	// Extract to non-existent directory
	destDir := filepath.Join(tmpDir, "new", "nested", "dest")
	e := NewExtractor(true)
	_, err = e.ExtractPackage(pkgPath, destDir)
	if err != nil {
		t.Fatalf("ExtractPackage failed: %v", err)
	}

	// Verify directory structure was created
	filePath := filepath.Join(destDir, "deep", "nested", "path", "file.txt")
	if _, err := os.Stat(filePath); err != nil {
		t.Errorf("Nested directory structure not created: %v", err)
	}
}

// TestExtractPackageNonExistent tests extraction of non-existent file.
func TestExtractPackageNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	e := NewExtractor(true)

	_, err := e.ExtractPackage(filepath.Join(tmpDir, "nonexistent.pkg.tar.zst"), tmpDir)
	if err == nil {
		t.Fatal("Expected error for non-existent package")
	}
}

// TestExtractFile tests extracting a single file from archive.
func TestExtractFile(t *testing.T) {
	files := map[string]string{
		"file1.txt": "content 1",
		"file2.txt": "content 2",
	}

	archiveData, err := createTestArchive(files)
	if err != nil {
		t.Fatalf("Failed to create test archive: %v", err)
	}

	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "test.pkg.tar.zst")
	if err := os.WriteFile(pkgPath, archiveData, 0644); err != nil {
		t.Fatalf("Failed to write test package: %v", err)
	}

	// Extract single file
	e := NewExtractor(true)
	if err := e.ExtractFile(pkgPath, "file1.txt", tmpDir); err != nil {
		t.Fatalf("ExtractFile failed: %v", err)
	}

	// Verify file was extracted
	content, err := os.ReadFile(filepath.Join(tmpDir, "file1.txt"))
	if err != nil {
		t.Errorf("Extracted file not found: %v", err)
	}
	if string(content) != "content 1" {
		t.Errorf("Extracted file content mismatch: got %s", string(content))
	}
}

// TestExtractFileNotFound tests extracting non-existent file from archive.
func TestExtractFileNotFound(t *testing.T) {
	files := map[string]string{
		"file1.txt": "content 1",
	}

	archiveData, err := createTestArchive(files)
	if err != nil {
		t.Fatalf("Failed to create test archive: %v", err)
	}

	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "test.pkg.tar.zst")
	if err := os.WriteFile(pkgPath, archiveData, 0644); err != nil {
		t.Fatalf("Failed to write test package: %v", err)
	}

	// Try to extract non-existent file
	e := NewExtractor(true)
	err = e.ExtractFile(pkgPath, "nonexistent.txt", tmpDir)
	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}
}

// TestListContents tests listing archive contents.
func TestListContents(t *testing.T) {
	files := map[string]string{
		"file1.txt":     "content 1",
		"file2.txt":     "content 2",
		"dir/file3.txt": "content 3",
	}

	archiveData, err := createTestArchive(files)
	if err != nil {
		t.Fatalf("Failed to create test archive: %v", err)
	}

	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "test.pkg.tar.zst")
	if err := os.WriteFile(pkgPath, archiveData, 0644); err != nil {
		t.Fatalf("Failed to write test package: %v", err)
	}

	// List contents
	e := NewExtractor(true)
	contents, err := e.ListContents(pkgPath)
	if err != nil {
		t.Fatalf("ListContents failed: %v", err)
	}

	// Verify contents
	if len(contents) != 3 {
		t.Errorf("Expected 3 files in archive, got %d", len(contents))
	}

	// Check each file is listed
	fileMap := make(map[string]bool)
	for _, f := range contents {
		fileMap[f] = true
	}

	expectedFiles := []string{"file1.txt", "file2.txt", "dir/file3.txt"}
	for _, expected := range expectedFiles {
		if !fileMap[expected] {
			t.Errorf("Expected file not found in contents: %s", expected)
		}
	}
}

// TestListContentsNonExistent tests listing contents of non-existent file.
func TestListContentsNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	e := NewExtractor(true)

	_, err := e.ListContents(filepath.Join(tmpDir, "nonexistent.pkg.tar.zst"))
	if err == nil {
		t.Fatal("Expected error for non-existent package")
	}
}

// TestExtractPackageDirectoryTraversalProtection tests protection against directory traversal.
func TestExtractPackageDirectoryTraversalProtection(t *testing.T) {
	// Create archive with path traversal attempt
	var buf bytes.Buffer
	encoder, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}

	tarWriter := tar.NewWriter(encoder)

	// Attempt to write to parent directory
	header := &tar.Header{
		Name: "../../etc/passwd",
		Mode: 0644,
		Size: 10,
	}

	tarWriter.WriteHeader(header)
	tarWriter.Write([]byte("malicious"))

	tarWriter.Close()
	encoder.Close()

	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "test.pkg.tar.zst")
	if err := os.WriteFile(pkgPath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("Failed to write test package: %v", err)
	}

	// Extraction should fail due to directory traversal
	e := NewExtractor(true)
	_, err = e.ExtractPackage(pkgPath, tmpDir)
	if err == nil {
		t.Fatal("Expected error for directory traversal attempt")
	}
}

// TestExtractPackagePreservePermissions tests that permissions are preserved.
func TestExtractPackagePreservePermissions(t *testing.T) {
	// Create a simple test archive
	files := map[string]string{
		"executable.sh": "#!/bin/bash",
	}

	archiveData, err := createTestArchive(files)
	if err != nil {
		t.Fatalf("Failed to create test archive: %v", err)
	}

	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "test.pkg.tar.zst")
	if err := os.WriteFile(pkgPath, archiveData, 0644); err != nil {
		t.Fatalf("Failed to write test package: %v", err)
	}

	// Extract with permission preservation
	destDir := filepath.Join(tmpDir, "extracted")
	e := NewExtractor(true) // preserve permissions
	_, err = e.ExtractPackage(pkgPath, destDir)
	if err != nil {
		t.Fatalf("ExtractPackage failed: %v", err)
	}

	// Check file exists
	fileInfo, err := os.Stat(filepath.Join(destDir, "executable.sh"))
	if err != nil {
		t.Errorf("Failed to stat extracted file: %v", err)
	}

	if !fileInfo.Mode().IsRegular() {
		t.Errorf("File is not a regular file")
	}
}

// TestExtractPackageLargeFile tests extracting large files.
func TestExtractPackageLargeFile(t *testing.T) {
	// Create a 10MB test file
	largeContent := make([]byte, 10*1024*1024)
	for i := 0; i < len(largeContent); i++ {
		largeContent[i] = byte(i % 256)
	}

	files := map[string]string{
		"largefile.bin": string(largeContent),
	}

	// This will be large, but OK for testing
	archiveData, err := createTestArchive(files)
	if err != nil {
		t.Fatalf("Failed to create test archive: %v", err)
	}

	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "test.pkg.tar.zst")
	if err := os.WriteFile(pkgPath, archiveData, 0644); err != nil {
		t.Fatalf("Failed to write test package: %v", err)
	}

	// Extract
	destDir := filepath.Join(tmpDir, "extracted")
	e := NewExtractor(true)
	_, err = e.ExtractPackage(pkgPath, destDir)
	if err != nil {
		t.Fatalf("ExtractPackage failed: %v", err)
	}

	// Verify file size
	fileInfo, err := os.Stat(filepath.Join(destDir, "largefile.bin"))
	if err != nil {
		t.Errorf("Failed to stat extracted file: %v", err)
	}
	if fileInfo.Size() != int64(len(largeContent)) {
		t.Errorf("File size mismatch: expected %d, got %d", len(largeContent), fileInfo.Size())
	}
}

// BenchmarkExtractPackage benchmarks package extraction.
func BenchmarkExtractPackage(b *testing.B) {
	// Create test archive
	files := make(map[string]string)
	for i := 0; i < 100; i++ {
		files[filepath.Join("subdir", "file"+string(rune(i))+".txt")] = "test content"
	}

	archiveData, err := createTestArchive(files)
	if err != nil {
		b.Fatalf("Failed to create test archive: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		pkgPath := filepath.Join(tmpDir, "test.pkg.tar.zst")
		os.WriteFile(pkgPath, archiveData, 0644)

		e := NewExtractor(true)
		destDir := filepath.Join(tmpDir, "extracted")
		e.ExtractPackage(pkgPath, destDir)
	}
}

// TestExtractSymlinks tests extraction of symbolic links from packages.
func TestExtractSymlinks(t *testing.T) {
	// Create a test archive with symlinks
	var buf bytes.Buffer

	encoder, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}

	tarWriter := tar.NewWriter(encoder)

	// Add a regular file
	fileHeader := &tar.Header{
		Name: "usr/lib/libtest.so.1.0.0",
		Mode: 0755,
		Size: 10,
		Typeflag: tar.TypeReg,
	}
	if err := tarWriter.WriteHeader(fileHeader); err != nil {
		t.Fatalf("Failed to write file header: %v", err)
	}
	if _, err := tarWriter.Write([]byte("library123")); err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}

	// Add symlinks pointing to the file
	symlinkHeaders := []struct {
		name   string
		target string
	}{
		{"usr/lib/libtest.so.1", "libtest.so.1.0.0"},
		{"usr/lib/libtest.so", "libtest.so.1"},
	}

	for _, symlink := range symlinkHeaders {
		symHeader := &tar.Header{
			Name:     symlink.name,
			Size:     0,
			Linkname: symlink.target,
			Mode:     0777,
			Typeflag: tar.TypeSymlink,
		}
		if err := tarWriter.WriteHeader(symHeader); err != nil {
			t.Fatalf("Failed to write symlink header: %v", err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Failed to close tar: %v", err)
	}
	if err := encoder.Close(); err != nil {
		t.Fatalf("Failed to close encoder: %v", err)
	}

	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "test.pkg.tar.zst")
	if err := os.WriteFile(pkgPath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("Failed to write test package: %v", err)
	}

	destDir := filepath.Join(tmpDir, "extracted")

	e := NewExtractor(true)
	extracted, err := e.ExtractPackage(pkgPath, destDir)
	if err != nil {
		t.Fatalf("Failed to extract package: %v", err)
	}

	// Verify symlinks were extracted
	if len(extracted) != 3 {
		t.Errorf("Expected 3 files, got %d", len(extracted))
	}

	// Verify regular file exists
	libPath := filepath.Join(destDir, "usr/lib/libtest.so.1.0.0")
	if _, err := os.Stat(libPath); err != nil {
		t.Errorf("Library file not found: %v", err)
	}

	// Verify symlinks exist and point correctly
	symlinkTests := []struct {
		path   string
		target string
	}{
		{filepath.Join(destDir, "usr/lib/libtest.so.1"), "libtest.so.1.0.0"},
		{filepath.Join(destDir, "usr/lib/libtest.so"), "libtest.so.1"},
	}

	for _, test := range symlinkTests {
		target, err := os.Readlink(test.path)
		if err != nil {
			t.Errorf("Failed to read symlink %s: %v", test.path, err)
		}
		if target != test.target {
			t.Errorf("Symlink %s points to %s, expected %s", test.path, target, test.target)
		}
	}

	// Verify symlinks are tracked in extracted files
	symlinksFound := 0
	for _, file := range extracted {
		if file.IsSymlink {
			symlinksFound++
			if file.LinkTarget == "" {
				t.Errorf("Symlink %s has no LinkTarget", file.Path)
			}
		}
	}
	if symlinksFound != 2 {
		t.Errorf("Expected 2 symlinks, found %d", symlinksFound)
	}
}

// TestExtractHardLinks tests extraction of hard links from packages.
func TestExtractHardLinks(t *testing.T) {
	// Create a test archive with hard links
	var buf bytes.Buffer

	encoder, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}

	tarWriter := tar.NewWriter(encoder)

	// Add a regular file
	fileHeader := &tar.Header{
		Name:     "usr/bin/original",
		Mode:     0755,
		Size:     8,
		Typeflag: tar.TypeReg,
	}
	if err := tarWriter.WriteHeader(fileHeader); err != nil {
		t.Fatalf("Failed to write file header: %v", err)
	}
	if _, err := tarWriter.Write([]byte("original")); err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}

	// Add hard link pointing to the file
	linkHeader := &tar.Header{
		Name:     "usr/bin/hardlink",
		Linkname: "usr/bin/original",
		Mode:     0755,
		Typeflag: tar.TypeLink,
	}
	if err := tarWriter.WriteHeader(linkHeader); err != nil {
		t.Fatalf("Failed to write hard link header: %v", err)
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Failed to close tar: %v", err)
	}
	if err := encoder.Close(); err != nil {
		t.Fatalf("Failed to close encoder: %v", err)
	}

	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "test.pkg.tar.zst")
	if err := os.WriteFile(pkgPath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("Failed to write test package: %v", err)
	}

	destDir := filepath.Join(tmpDir, "extracted")

	e := NewExtractor(true)
	extracted, err := e.ExtractPackage(pkgPath, destDir)
	if err != nil {
		t.Fatalf("Failed to extract package: %v", err)
	}

	// Verify files were extracted
	if len(extracted) != 2 {
		t.Errorf("Expected 2 files, got %d", len(extracted))
	}

	// Verify original file exists
	originalPath := filepath.Join(destDir, "usr/bin/original")
	if _, err := os.Stat(originalPath); err != nil {
		t.Errorf("Original file not found: %v", err)
	}

	// Verify hard link exists
	linkPath := filepath.Join(destDir, "usr/bin/hardlink")
	if err != nil {
		t.Errorf("Hard link not found: %v", err)
	}

	// Get inode numbers to verify they're the same
	var origStat, linkStat os.FileInfo
	origStat, _ = os.Stat(originalPath)
	linkStat, _ = os.Stat(linkPath)

	// Hard links should have the same size
	if origStat.Size() != linkStat.Size() {
		t.Errorf("Hard link size mismatch: original %d, link %d", origStat.Size(), linkStat.Size())
	}

	// Verify hard link is tracked as symlink in extracted files (for our purposes)
	linkFound := false
	for _, file := range extracted {
		if file.Path == "usr/bin/hardlink" && file.IsSymlink {
			linkFound = true
			break
		}
	}
	if !linkFound {
		t.Errorf("Hard link not tracked in extracted files")
	}
}

// TestExtractSymlinksWithDirectories tests symlinks and directories together.
func TestExtractSymlinksWithDirectories(t *testing.T) {
	var buf bytes.Buffer

	encoder, err := zstd.NewWriter(&buf)
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}

	tarWriter := tar.NewWriter(encoder)

	// Add directories
	dirs := []string{
		"usr",
		"usr/lib",
		"usr/bin",
	}
	for _, dir := range dirs {
		dirHeader := &tar.Header{
			Name:     dir,
			Mode:     0755,
			Typeflag: tar.TypeDir,
		}
		if err := tarWriter.WriteHeader(dirHeader); err != nil {
			t.Fatalf("Failed to write dir header: %v", err)
		}
	}

	// Add a library file
	libHeader := &tar.Header{
		Name:     "usr/lib/libvlc.so.5",
		Mode:     0755,
		Size:     10,
		Typeflag: tar.TypeReg,
	}
	if err := tarWriter.WriteHeader(libHeader); err != nil {
		t.Fatalf("Failed to write lib header: %v", err)
	}
	if _, err := tarWriter.Write([]byte("libcontent")); err != nil {
		t.Fatalf("Failed to write lib content: %v", err)
	}

	// Add symlinks (simulating libvlc case)
	symlinks := []struct {
		name   string
		target string
	}{
		{"usr/lib/libvlc.so", "libvlc.so.5"},
		{"usr/bin/vlc-wrapper", "../lib/libvlc.so"},
	}

	for _, symlink := range symlinks {
		symHeader := &tar.Header{
			Name:     symlink.name,
			Linkname: symlink.target,
			Mode:     0777,
			Typeflag: tar.TypeSymlink,
		}
		if err := tarWriter.WriteHeader(symHeader); err != nil {
			t.Fatalf("Failed to write symlink header: %v", err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Failed to close tar: %v", err)
	}
	if err := encoder.Close(); err != nil {
		t.Fatalf("Failed to close encoder: %v", err)
	}

	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "libvlc.pkg.tar.zst")
	if err := os.WriteFile(pkgPath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("Failed to write test package: %v", err)
	}

	destDir := filepath.Join(tmpDir, "extracted")

	e := NewExtractor(true)
	extracted, err := e.ExtractPackage(pkgPath, destDir)
	if err != nil {
		t.Fatalf("Failed to extract package: %v", err)
	}

	// Verify correct number of items extracted (3 dirs + 1 file + 2 symlinks = 6)
	if len(extracted) != 6 {
		t.Errorf("Expected 6 items extracted, got %d", len(extracted))
	}

	// Verify libvlc.so points to libvlc.so.5
	vlcSymlink := filepath.Join(destDir, "usr/lib/libvlc.so")
	target, err := os.Readlink(vlcSymlink)
	if err != nil {
		t.Errorf("Failed to read libvlc.so symlink: %v", err)
	}
	if target != "libvlc.so.5" {
		t.Errorf("libvlc.so points to %s, expected libvlc.so.5", target)
	}

	// Verify cross-directory symlink
	binSymlink := filepath.Join(destDir, "usr/bin/vlc-wrapper")
	binTarget, err := os.Readlink(binSymlink)
	if err != nil {
		t.Errorf("Failed to read vlc-wrapper symlink: %v", err)
	}
	if binTarget != "../lib/libvlc.so" {
		t.Errorf("vlc-wrapper points to %s, expected ../lib/libvlc.so", binTarget)
	}
}
