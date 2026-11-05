package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"torbox/fileshare/core"
)

type DiskValue struct {
	Value uint64 `json:"value"`
	Text  string `json:"text"`
}

type DiskInfoResponse struct {
	Used      DiskValue `json:"used"`
	Available DiskValue `json:"available"`
	Total     DiskValue `json:"total"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type UploadResponse struct {
	SuccessMessages []string `json:"success_messages"`
	ErrorMessages   []string `json:"error_messages"`
}

type Handlers struct {
	MediaRoot   string
	BaseDir     string
	OnionDomain string
}

func NewHandlers(mediaRoot, baseDir, onionDomain string) *Handlers {
	return &Handlers{
		MediaRoot:   mediaRoot,
		BaseDir:     baseDir,
		OnionDomain: onionDomain,
	}
}

func (h *Handlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handlers) writeError(w http.ResponseWriter, status int, msg string) {
	h.writeJSON(w, status, ErrorResponse{Error: msg})
}

// GetDiskInfo handles GET /get_disk_info.
func (h *Handlers) HandleGetDiskInfo(w http.ResponseWriter, r *http.Request) {
	total, free, used, err := core.GetDiskUsage(h.MediaRoot)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Could not obtain disk info")
		return
	}

	resp := DiskInfoResponse{
		Used: DiskValue{
			Value: used,
			Text:  core.FormatSize(int64(used)),
		},
		Available: DiskValue{
			Value: free,
			Text:  core.FormatSize(int64(free)),
		},
		Total: DiskValue{
			Value: total,
			Text:  core.FormatSize(int64(total)),
		},
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// extractPathQueryParam safely extracts the "path" query parameter from an HTTP request.
// It unescapes percent-encoded sequences (such as %20, %2B, %5B, %5D) while preserving
// literal '+' characters without erroneously converting them to spaces.
func extractPathQueryParam(r *http.Request) string {
	rawQuery := r.URL.RawQuery
	for rawQuery != "" {
		var part string
		part, rawQuery, _ = strings.Cut(rawQuery, "&")
		if part == "" {
			continue
		}
		k, v, _ := strings.Cut(part, "=")
		if k == "path" {
			if unescaped, err := url.PathUnescape(v); err == nil {
				return strings.ReplaceAll(unescaped, "//", "/")
			}
			return strings.ReplaceAll(v, "//", "/")
		}
	}
	raw := r.URL.Query().Get("path")
	return strings.ReplaceAll(raw, "//", "/")
}

// HandleGetTree handles GET /get_tree?path=<path>.
func (h *Handlers) HandleGetTree(w http.ResponseWriter, r *http.Request) {
	reqPath := extractPathQueryParam(r)
	resp, err := core.ListFiles(h.MediaRoot, reqPath)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid directory path")
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// HandleUploadFiles handles POST /upload_files?path=<path>.
func (h *Handlers) HandleUploadFiles(w http.ResponseWriter, r *http.Request) {
	uploadPath := extractPathQueryParam(r)
	if strings.Contains(uploadPath, "..") {
		h.writeError(w, http.StatusBadRequest, "Invalid directory specified")
		return
	}

	targetDir, err := core.ResolveSafePath(h.MediaRoot, uploadPath)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid directory specified")
		return
	}

	info, err := os.Stat(targetDir)
	if (err != nil || !info.IsDir()) && strings.Contains(uploadPath, "+") {
		if altPath, errAlt := core.ResolveSafePath(h.MediaRoot, strings.ReplaceAll(uploadPath, "+", " ")); errAlt == nil {
			if altInfo, errStat := os.Stat(altPath); errStat == nil && altInfo.IsDir() {
				targetDir = altPath
				info = altInfo
				err = nil
			}
		}
	}
	if err != nil || !info.IsDir() {
		h.writeError(w, http.StatusBadRequest, "Invalid directory specified")
		return
	}

	if !core.HasPermissions(h.MediaRoot, targetDir, "w") {
		h.writeError(w, http.StatusBadRequest, "Insufficient permissions to upload to this directory")
		return
	}

	// Max 100MB in memory, rest to temp files
	err = r.ParseMultipartForm(100 << 20)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	var files = r.MultipartForm.File["files[]"]
	if len(files) == 0 {
		files = r.MultipartForm.File["files"]
	}

	if len(files) == 0 {
		h.writeJSON(w, http.StatusOK, map[string]string{"error": "No files selected"})
		return
	}

	var successMessages []string
	var errorMessages []string

	for _, fileHeader := range files {
		if fileHeader.Filename == "" {
			errorMessages = append(errorMessages, "No selected file")
			continue
		}

		baseName := filepath.Base(fileHeader.Filename)
		targetFile := filepath.Join(targetDir, baseName)

		if _, err := os.Stat(targetFile); err == nil {
			ext := filepath.Ext(baseName)
			nameOnly := strings.TrimSuffix(baseName, ext)
			counter := 1
			for {
				candidate := fmt.Sprintf("%s (%d)%s", nameOnly, counter, ext)
				candidatePath := filepath.Join(targetDir, candidate)
				if _, err := os.Stat(candidatePath); os.IsNotExist(err) {
					targetFile = candidatePath
					break
				}
				counter++
			}
		}

		src, err := fileHeader.Open()
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("Failed to read file %s", fileHeader.Filename))
			continue
		}

		dst, err := os.Create(targetFile)
		if err != nil {
			src.Close()
			errorMessages = append(errorMessages, fmt.Sprintf("Failed to save file %s", fileHeader.Filename))
			continue
		}

		_, err = io.Copy(dst, src)
		src.Close()
		dst.Close()

		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("Failed writing file %s", fileHeader.Filename))
			continue
		}

		successMessages = append(successMessages, fmt.Sprintf("File \"%s\" uploaded successfully", fileHeader.Filename))
	}

	if successMessages == nil {
		successMessages = []string{}
	}
	if errorMessages == nil {
		errorMessages = []string{}
	}

	h.writeJSON(w, http.StatusOK, UploadResponse{
		SuccessMessages: successMessages,
		ErrorMessages:   errorMessages,
	})
}

// HandleDownload handles GET /download?path=<path>.
func (h *Handlers) HandleDownload(w http.ResponseWriter, r *http.Request) {
	rawPath := extractPathQueryParam(r)

	filePath, err := core.ResolveSafePath(h.MediaRoot, rawPath)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid file specified")
		return
	}

	info, err := os.Stat(filePath)
	if (err != nil || info.IsDir()) && strings.Contains(rawPath, "+") {
		if altPath, errAlt := core.ResolveSafePath(h.MediaRoot, strings.ReplaceAll(rawPath, "+", " ")); errAlt == nil {
			if altInfo, errStat := os.Stat(altPath); errStat == nil && !altInfo.IsDir() {
				filePath = altPath
				info = altInfo
				err = nil
			}
		}
	}
	if err != nil || info.IsDir() {
		h.writeError(w, http.StatusBadRequest, "Invalid file specified")
		return
	}

	parentDir := filepath.Dir(filePath)
	if !core.HasPermissions(h.MediaRoot, parentDir, "r") {
		h.writeError(w, http.StatusBadRequest, "Insufficient permissions to download this file")
		return
	}

	fileName := filepath.Base(filePath)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	http.ServeFile(w, r, filePath)
}

// HandleDownloadFolder handles GET /download_folder?path=<path>&check=<check>.
func (h *Handlers) HandleDownloadFolder(w http.ResponseWriter, r *http.Request) {
	rawPath := extractPathQueryParam(r)

	folderPath, err := core.ResolveSafePath(h.MediaRoot, rawPath)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid folder specified")
		return
	}

	info, err := os.Stat(folderPath)
	if (err != nil || !info.IsDir()) && strings.Contains(rawPath, "+") {
		if altPath, errAlt := core.ResolveSafePath(h.MediaRoot, strings.ReplaceAll(rawPath, "+", " ")); errAlt == nil {
			if altInfo, errStat := os.Stat(altPath); errStat == nil && altInfo.IsDir() {
				folderPath = altPath
				info = altInfo
				err = nil
			}
		}
	}
	if err != nil || !info.IsDir() {
		h.writeError(w, http.StatusBadRequest, "Invalid folder specified")
		return
	}

	if !core.HasPermissions(h.MediaRoot, folderPath, "r") {
		h.writeError(w, http.StatusBadRequest, "Insufficient permissions to download this folder")
		return
	}

	// Check disk space
	folderSize := core.GetDirectorySize(h.MediaRoot, folderPath)
	_, freeSpace, _, _ := core.GetDiskUsage(h.MediaRoot)

	if folderSize > int64(freeSpace) {
		h.writeError(w, http.StatusBadRequest, "Insufficient space to create the zip file")
		return
	}

	check := r.URL.Query().Get("check")
	if check != "" {
		h.writeJSON(w, http.StatusOK, MessageResponse{Message: "Enough space to create the zip file"})
		return
	}

	folderName := filepath.Base(folderPath)
	if folderName == "" || folderName == "." || folderName == "/" || filepath.Clean(folderPath) == filepath.Clean(h.MediaRoot) {
		folderName = "root"
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", folderName))

	err = core.CompressFolderToZip(folderPath, w)
	if err != nil {
		// Log or handle error if headers haven't sent, but streaming might have begun
		return
	}
}

// HandleIndex serves webclient/dist/index.html.
func (h *Handlers) HandleIndex(w http.ResponseWriter, r *http.Request) {
	indexPath := filepath.Join(h.BaseDir, "webclient", "dist", "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, indexPath)
}
