package core

import (
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	ErrInvalidDirectory = errors.New("Invalid directory specified")
	ErrInvalidPath      = errors.New("Invalid directory path")
	ErrInvalidFile      = errors.New("Invalid file specified")
	ErrPermissionDenied = errors.New("Insufficient permissions")
)

type CurrentDirInfo struct {
	Path           string `json:"path"`
	ParentPath     string `json:"parent_path"`
	Permissions    string `json:"permissions"`
	Size           string `json:"size"`
	SizeBytes      int64  `json:"size_bytes"`
	TotalFileCount int    `json:"total_file_count"`
}

type ItemData struct {
	Name      string  `json:"name"`
	Type      *string `json:"type"`
	Path      string  `json:"path,omitempty"`
	Size      string  `json:"size"`
	SizeBytes int64   `json:"size_bytes"`
}

type TreeItem struct {
	Data ItemData `json:"data"`
}

type TreeResponse struct {
	Current CurrentDirInfo `json:"current"`
	Tree    []TreeItem     `json:"tree"`
}

// FormatSize formats a byte count into a human-readable string (B, KB, MB, GB, TB).
func FormatSize(size int64) string {
	fSize := float64(size)
	units := []string{"B", "KB", "MB", "GB", "TB"}
	var unit string
	for _, u := range units {
		unit = u
		if fSize < 1024.0 {
			break
		}
		fSize /= 1024.0
	}
	return fmt.Sprintf("%.2f %s", fSize, unit)
}

// GetDirectorySize calculates the total size of files inside dirPath recursively.
// Returns 0 if directory does not have execute ('x') permission.
func GetDirectorySize(mediaRoot, dirPath string) int64 {
	if !HasPermissions(mediaRoot, dirPath, "x") {
		return 0
	}

	var totalSize int64
	_ = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	return totalSize
}

// GetFileType determines the type category from MIME (e.g. "application", "image", "text").
func GetFileType(filePath string) *string {
	ext := filepath.Ext(filePath)
	if ext == "" {
		return nil
	}
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return nil
	}
	parts := strings.Split(mimeType, ";")
	typePart := strings.TrimSpace(parts[0])
	slashParts := strings.Split(typePart, "/")
	if len(slashParts) > 0 && slashParts[0] != "" {
		val := slashParts[0]
		return &val
	}
	return nil
}

// ResolveSafePath safely resolves a relative path within mediaRoot, preventing directory traversal.
func ResolveSafePath(mediaRoot, reqPath string) (string, error) {
	// Reject explicit traversal attempts
	if strings.Contains(reqPath, "..") {
		return "", ErrInvalidPath
	}

	// Clean for double slashes and convert
	cleanReq := filepath.Clean(strings.ReplaceAll(reqPath, "//", "/"))
	if cleanReq == "/" || cleanReq == "." || cleanReq == "" {
		return mediaRoot, nil
	}

	// Remove leading slashes for join
	rel := strings.TrimPrefix(cleanReq, "/")
	if rel == "" {
		return mediaRoot, nil
	}

	fullPath := filepath.Join(mediaRoot, rel)
	cleanFull := filepath.Clean(fullPath)
	cleanRoot := filepath.Clean(mediaRoot)

	relToRoot, err := filepath.Rel(cleanRoot, cleanFull)
	if err != nil || strings.HasPrefix(relToRoot, "..") || relToRoot == ".." {
		return "", ErrInvalidPath
	}

	return cleanFull, nil
}

// GetDirectoryTree lists folders then files in dirPath.
func GetDirectoryTree(mediaRoot, dirPath string) ([]TreeItem, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var folderNames []string
	var fileNames []string

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == AccessFile {
			continue
		}
		if entry.IsDir() {
			folderNames = append(folderNames, name)
		} else {
			fileNames = append(fileNames, name)
		}
	}

	sort.Strings(folderNames)
	sort.Strings(fileNames)

	var items []TreeItem
	folderType := "Folder"

	for _, f := range folderNames {
		subDirPath := filepath.Join(dirPath, f)
		relPath, _ := filepath.Rel(mediaRoot, subDirPath)
		sizeBytes := GetDirectorySize(mediaRoot, subDirPath)
		items = append(items, TreeItem{
			Data: ItemData{
				Name:      f,
				Type:      &folderType,
				Path:      relPath,
				Size:      FormatSize(sizeBytes),
				SizeBytes: sizeBytes,
			},
		})
	}

	for _, f := range fileNames {
		filePath := filepath.Join(dirPath, f)
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}
		sizeBytes := info.Size()
		items = append(items, TreeItem{
			Data: ItemData{
				Name:      f,
				Type:      GetFileType(filePath),
				Size:      FormatSize(sizeBytes),
				SizeBytes: sizeBytes,
			},
		})
	}

	if items == nil {
		items = []TreeItem{}
	}

	return items, nil
}

// ListFiles builds the TreeResponse for a requested directory path.
func ListFiles(mediaRoot, reqPath string) (*TreeResponse, error) {
	dirPath, err := ResolveSafePath(mediaRoot, reqPath)
	if err != nil {
		return nil, ErrInvalidPath
	}

	info, err := os.Stat(dirPath)
	if (err != nil || !info.IsDir()) && strings.Contains(reqPath, "+") {
		if altPath, errAlt := ResolveSafePath(mediaRoot, strings.ReplaceAll(reqPath, "+", " ")); errAlt == nil {
			if altInfo, errStat := os.Stat(altPath); errStat == nil && altInfo.IsDir() {
				dirPath = altPath
				info = altInfo
				err = nil
			}
		}
	}
	if err != nil || !info.IsDir() {
		return nil, ErrInvalidPath
	}

	perms := GetPermissions(mediaRoot, dirPath)

	var files []TreeItem
	if strings.ContainsRune(perms, 'x') {
		tree, err := GetDirectoryTree(mediaRoot, dirPath)
		if err == nil {
			files = tree
		} else {
			files = []TreeItem{}
		}
	} else {
		files = []TreeItem{}
	}

	// Count direct items excluding hidden and .access
	entries, _ := os.ReadDir(dirPath)
	totalCount := 0
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, ".") && name != AccessFile {
			totalCount++
		}
	}

	// Calculate current path relative to mediaRoot
	relPath, _ := filepath.Rel(mediaRoot, dirPath)
	curPath := relPath
	if curPath == "." {
		curPath = "/"
	}

	// Calculate parent path relative to mediaRoot
	var parentPath string
	if filepath.Clean(dirPath) == filepath.Clean(mediaRoot) {
		parentPath = "/"
	} else {
		parentRel, _ := filepath.Rel(mediaRoot, filepath.Dir(dirPath))
		if parentRel == "." {
			parentPath = "/"
		} else {
			parentPath = parentRel
		}
	}

	dirSizeBytes := GetDirectorySize(mediaRoot, dirPath)

	return &TreeResponse{
		Current: CurrentDirInfo{
			Path:           curPath,
			ParentPath:     parentPath,
			Permissions:    perms,
			Size:           FormatSize(dirSizeBytes),
			SizeBytes:      dirSizeBytes,
			TotalFileCount: totalCount,
		},
		Tree: files,
	}, nil
}
