package core

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0.00 B"},
		{500, "500.00 B"},
		{1024, "1.00 KB"},
		{10695475, "10.20 MB"},
		{5347737, "5.10 MB"},
		{14748364800, "13.74 GB"},
	}

	for _, tt := range tests {
		got := FormatSize(tt.bytes)
		if got != tt.expected {
			t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.expected)
		}
	}
}

func TestPermissions(t *testing.T) {
	tempDir := t.TempDir()
	pubDir := filepath.Join(tempDir, "public")
	readDir := filepath.Join(tempDir, "readonly")
	noPermDir := filepath.Join(tempDir, "other")

	_ = os.MkdirAll(pubDir, 0755)
	_ = os.MkdirAll(readDir, 0755)
	_ = os.MkdirAll(noPermDir, 0755)

	accessContent := pubDir + ";rwx\n" + readDir + ";r\n"
	err := os.WriteFile(filepath.Join(tempDir, AccessFile), []byte(accessContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write .access: %v", err)
	}

	if !HasPermissions(tempDir, pubDir, "rwx") {
		t.Errorf("Expected pubDir to have rwx")
	}
	if !HasPermissions(tempDir, pubDir, "w") {
		t.Errorf("Expected pubDir to have w")
	}
	if HasPermissions(tempDir, readDir, "w") {
		t.Errorf("Expected readDir to NOT have w")
	}
	if !HasPermissions(tempDir, readDir, "r") {
		t.Errorf("Expected readDir to have r")
	}

	// Default perms for unlisted should be "rx"
	if GetPermissions(tempDir, noPermDir) != "rx" {
		t.Errorf("Expected default perms rx for %s, got %s", noPermDir, GetPermissions(tempDir, noPermDir))
	}
}

func TestResolveSafePath(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "sub")
	_ = os.MkdirAll(subDir, 0755)

	// Valid paths
	p, err := ResolveSafePath(tempDir, "")
	if err != nil || p != tempDir {
		t.Errorf("Expected %s, got %s, err: %v", tempDir, p, err)
	}

	p, err = ResolveSafePath(tempDir, "/")
	if err != nil || p != tempDir {
		t.Errorf("Expected %s, got %s, err: %v", tempDir, p, err)
	}

	p, err = ResolveSafePath(tempDir, "/sub")
	if err != nil || p != subDir {
		t.Errorf("Expected %s, got %s, err: %v", subDir, p, err)
	}

	// Traversal attacks
	_, err = ResolveSafePath(tempDir, "../")
	if err == nil {
		t.Errorf("Expected error on traversal attempt")
	}

	_, err = ResolveSafePath(tempDir, "/sub/../../etc")
	if err == nil {
		t.Errorf("Expected error on traversal attempt")
	}
}

func TestListFilesAndZip(t *testing.T) {
	tempDir := t.TempDir()
	docDir := filepath.Join(tempDir, "docs")
	_ = os.MkdirAll(docDir, 0755)

	filePath := filepath.Join(docDir, "test.txt")
	_ = os.WriteFile(filePath, []byte("hello world"), 0644)

	res, err := ListFiles(tempDir, "/")
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if res.Current.Path != "/" {
		t.Errorf("Expected current.path to be '/', got %s", res.Current.Path)
	}
	if len(res.Tree) != 1 {
		t.Fatalf("Expected 1 item in tree, got %d", len(res.Tree))
	}
	if res.Tree[0].Data.Name != "docs" || *res.Tree[0].Data.Type != "Folder" {
		t.Errorf("Unexpected tree item: %+v", res.Tree[0].Data)
	}

	// Test ZIP
	var buf bytes.Buffer
	err = CompressFolderToZip(docDir, &buf)
	if err != nil {
		t.Fatalf("CompressFolderToZip failed: %v", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("Failed to open zip reader: %v", err)
	}

	found := false
	for _, f := range zipReader.File {
		if strings.Contains(f.Name, "test.txt") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("test.txt not found in ZIP archive")
	}
}
