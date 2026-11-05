package core

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const AccessFile = ".access"
const DefaultPermissions = "rx"

// GetPermissions reads the .access file in mediaRoot and finds permissions for dirPath.
// If the path is not found or .access does not exist, it returns "rx".
func GetPermissions(mediaRoot, dirPath string) string {
	accessFilePath := filepath.Join(mediaRoot, AccessFile)
	file, err := os.Open(accessFilePath)
	if err != nil {
		return DefaultPermissions
	}
	defer file.Close()

	cleanDir := filepath.Clean(dirPath)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ";", 2)
		if len(parts) != 2 {
			continue
		}
		rulePath := strings.TrimSpace(parts[0])
		perms := strings.TrimSpace(parts[1])

		// Match absolute path or relative path
		if filepath.Clean(rulePath) == cleanDir {
			return perms
		}
		if !filepath.IsAbs(rulePath) && filepath.Clean(filepath.Join(mediaRoot, rulePath)) == cleanDir {
			return perms
		}
	}

	return DefaultPermissions
}

// HasPermissions checks if the directory has all characters in permissionRequired.
func HasPermissions(mediaRoot, dirPath, permissionRequired string) bool {
	perms := GetPermissions(mediaRoot, dirPath)
	for _, ch := range permissionRequired {
		if !strings.ContainsRune(perms, ch) {
			return false
		}
	}
	return true
}
