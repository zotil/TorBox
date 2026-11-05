package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"torbox/fileshare/core"
)

func setupTestEnvironment(t *testing.T) (string, string) {
	tempDir := t.TempDir()
	mediaRoot := filepath.Join(tempDir, "media")
	baseDir := filepath.Join(tempDir, "base")

	_ = os.MkdirAll(mediaRoot, 0755)
	_ = os.MkdirAll(filepath.Join(baseDir, "webclient", "dist", "assets"), 0755)

	_ = os.WriteFile(filepath.Join(baseDir, "webclient", "dist", "index.html"), []byte("<html><body>TorBox</body></html>"), 0644)
	_ = os.WriteFile(filepath.Join(baseDir, "webclient", "dist", "assets", "style.css"), []byte("body { color: red; }"), 0644)

	// Create test file in mediaRoot
	_ = os.WriteFile(filepath.Join(mediaRoot, "test.txt"), []byte("file content"), 0644)

	// Create .access giving rwx to mediaRoot
	accessContent := mediaRoot + ";rwx\n"
	_ = os.WriteFile(filepath.Join(mediaRoot, core.AccessFile), []byte(accessContent), 0644)

	return mediaRoot, baseDir
}

func TestGetDiskInfo(t *testing.T) {
	mediaRoot, baseDir := setupTestEnvironment(t)
	handler := NewServer(mediaRoot, baseDir, "example.onion")

	req := httptest.NewRequest(http.MethodGet, "/get_disk_info", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var resp DiskInfoResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if resp.Total.Value == 0 || resp.Total.Text == "" {
		t.Errorf("Expected valid total disk info, got %+v", resp.Total)
	}
}

func TestGetTree(t *testing.T) {
	mediaRoot, baseDir := setupTestEnvironment(t)
	handler := NewServer(mediaRoot, baseDir, "example.onion")

	req := httptest.NewRequest(http.MethodGet, "/get_tree?path=/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var resp core.TreeResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if resp.Current.Path != "/" {
		t.Errorf("Expected current path '/', got '%s'", resp.Current.Path)
	}
	if len(resp.Tree) != 1 {
		t.Fatalf("Expected 1 item in tree, got %d", len(resp.Tree))
	}
	if resp.Tree[0].Data.Name != "test.txt" {
		t.Errorf("Expected test.txt, got %s", resp.Tree[0].Data.Name)
	}

	// Traversal attempt
	req = httptest.NewRequest(http.MethodGet, "/get_tree?path=../", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request on traversal, got %d", w.Code)
	}
}

func TestGetTreeSpecialCharacters(t *testing.T) {
	mediaRoot, baseDir := setupTestEnvironment(t)
	handler := NewServer(mediaRoot, baseDir, "example.onion")

	folderName := "Cosmic Explorer 3 Enhanced Edition [1080p][AC3 5.1 Audio1-DTS-HD 5.1  Audio2+Subs][Track-A]"
	specialDir := filepath.Join(mediaRoot, folderName)
	err := os.MkdirAll(specialDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create special dir: %v", err)
	}
	_ = os.WriteFile(filepath.Join(specialDir, "movie.mkv"), []byte("video content"), 0644)

	// Simulate exact URL browser sends
	rawQuery := "path=Cosmic%20Explorer%203%20Enhanced%20Edition%20[1080p][AC3%205.1%20Audio1-DTS-HD%205.1%20%20Audio2+Subs][Track-A]"
	req := httptest.NewRequest(http.MethodGet, "/get_tree?"+rawQuery, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for special characters, got %d (body: %s)", w.Code, w.Body.String())
	}

	var resp core.TreeResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	if len(resp.Tree) != 1 || resp.Tree[0].Data.Name != "movie.mkv" {
		t.Errorf("Expected movie.mkv in tree, got %+v", resp.Tree)
	}

	// Test download file inside special character folder
	downPath := strings.ReplaceAll(folderName+"/movie.mkv", " ", "%20")
	downReq := httptest.NewRequest(http.MethodGet, "/download?path="+downPath, nil)
	wDown := httptest.NewRecorder()
	handler.ServeHTTP(wDown, downReq)
	if wDown.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK downloading file in special dir, got %d", wDown.Code)
	}

	// Test download folder as zip
	zipPath := strings.ReplaceAll(folderName, " ", "%20")
	zipReq := httptest.NewRequest(http.MethodGet, "/download_folder?path="+zipPath, nil)
	wZip := httptest.NewRecorder()
	handler.ServeHTTP(wZip, zipReq)
	if wZip.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK downloading zip of special dir, got %d", wZip.Code)
	}
}

func TestUploadAndDownload(t *testing.T) {
	mediaRoot, baseDir := setupTestEnvironment(t)
	handler := NewServer(mediaRoot, baseDir, "example.onion")

	// 1. Upload
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("files[]", "upload.txt")
	if err != nil {
		t.Fatalf("CreateFormFile failed: %v", err)
	}
	_, _ = part.Write([]byte("uploaded content"))
	_ = writer.Close()

	uploadReq := httptest.NewRequest(http.MethodPost, "/upload_files?path=/", body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, uploadReq)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on upload, got %d, body: %s", w.Code, w.Body.String())
	}

	var uploadResp UploadResponse
	_ = json.Unmarshal(w.Body.Bytes(), &uploadResp)
	if len(uploadResp.SuccessMessages) != 1 {
		t.Fatalf("Expected 1 success message, got %d", len(uploadResp.SuccessMessages))
	}

	// 2. Download
	downReq := httptest.NewRequest(http.MethodGet, "/download?path=upload.txt", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, downReq)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on download, got %d", w.Code)
	}
	if w.Body.String() != "uploaded content" {
		t.Errorf("Expected 'uploaded content', got '%s'", w.Body.String())
	}

	// 3. Download traversal
	downReq = httptest.NewRequest(http.MethodGet, "/download?path=../../etc/passwd", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, downReq)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request on download traversal, got %d", w.Code)
	}
}

func TestDownloadFolder(t *testing.T) {
	mediaRoot, baseDir := setupTestEnvironment(t)
	handler := NewServer(mediaRoot, baseDir, "example.onion")

	// Check free space
	req := httptest.NewRequest(http.MethodGet, "/download_folder?path=/&check=1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on check, got %d", w.Code)
	}

	// Actual download
	req = httptest.NewRequest(http.MethodGet, "/download_folder?path=/", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on folder download, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/zip" {
		t.Errorf("Expected application/zip, got %s", w.Header().Get("Content-Type"))
	}
}

func TestStaticFiles(t *testing.T) {
	mediaRoot, baseDir := setupTestEnvironment(t)
	handler := NewServer(mediaRoot, baseDir, "example.onion")

	// Index
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for /, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("TorBox")) {
		t.Errorf("Expected TorBox in index body, got %s", w.Body.String())
	}

	// Assets
	req = httptest.NewRequest(http.MethodGet, "/assets/style.css", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for /assets/style.css, got %d", w.Code)
	}
}

func TestCORSAndGzip(t *testing.T) {
	mediaRoot, baseDir := setupTestEnvironment(t)
	handler := NewServer(mediaRoot, baseDir, "example.onion")

	// Preflight OPTIONS
	req := httptest.NewRequest(http.MethodOptions, "/get_tree", nil)
	req.Header.Set("Origin", "http://example.onion")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for OPTIONS, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.onion" {
		t.Errorf("Expected CORS origin http://example.onion, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Private-Network") != "true" {
		t.Errorf("Expected Allow-Private-Network true")
	}

	// Gzip request
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("Expected Content-Encoding gzip, got %s", w.Header().Get("Content-Encoding"))
	}
}
