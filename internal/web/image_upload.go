package web

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const maxImageUploadBytes = 10 << 20

type imageUploads struct {
	mu    sync.Mutex
	paths map[string][]string
}

func newImageUploads() *imageUploads {
	uploads := &imageUploads{paths: make(map[string][]string)}
	uploads.removeExpired()
	return uploads
}

func (uploads *imageUploads) save(terminalID string, file io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(file, maxImageUploadBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) == 0 || len(data) > maxImageUploadBytes {
		return "", fmt.Errorf("image upload exceeds size limit")
	}
	contentType := http.DetectContentType(data[:min(len(data), 512)])
	extension := imageExtension(contentType)
	if extension == "" {
		return "", fmt.Errorf("unsupported image type")
	}
	directory, err := os.MkdirTemp("", "code-remote-image-")
	if err != nil {
		return "", err
	}
	path := filepath.Join(directory, "upload"+extension)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		_ = os.RemoveAll(directory)
		return "", err
	}

	uploads.mu.Lock()
	uploads.paths[terminalID] = append(uploads.paths[terminalID], directory)
	uploads.mu.Unlock()
	time.AfterFunc(24*time.Hour, func() {
		_ = os.RemoveAll(directory)
	})
	return path, nil
}

func (uploads *imageUploads) removeExpired() {
	directories, err := filepath.Glob(filepath.Join(os.TempDir(), "code-remote-image-*"))
	if err != nil {
		return
	}
	expiredBefore := time.Now().Add(-24 * time.Hour)
	for _, directory := range directories {
		info, err := os.Stat(directory)
		if err == nil && info.ModTime().Before(expiredBefore) {
			_ = os.RemoveAll(directory)
		}
	}
}

func (uploads *imageUploads) removeTerminal(terminalID string) {
	uploads.mu.Lock()
	directories := uploads.paths[terminalID]
	delete(uploads.paths, terminalID)
	uploads.mu.Unlock()
	for _, directory := range directories {
		_ = os.RemoveAll(directory)
	}
}

func imageExtension(contentType string) string {
	switch strings.Split(contentType, ";")[0] {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}
