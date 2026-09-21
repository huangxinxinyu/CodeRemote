// Package directory validates and browses working directories without creating
// project records or scanning recursively.
package directory

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxDirectories = 200

// ErrInvalid is returned when a submitted path is not a usable directory.
var ErrInvalid = errors.New("invalid directory")

// Entry is one direct child directory.
type Entry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Listing is a bounded, non-recursive directory response.
type Listing struct {
	Path        string  `json:"path"`
	Parent      string  `json:"parent"`
	Directories []Entry `json:"directories"`
}

// Resolve expands the current user's home shorthand and returns a canonical,
// existing directory path. An empty request uses fallback.
func Resolve(requested, fallback string) (string, error) {
	workingDir := strings.TrimSpace(requested)
	if workingDir == "" {
		workingDir = fallback
	}
	if workingDir == "~" || strings.HasPrefix(workingDir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("%w: resolve home directory", ErrInvalid)
		}
		if workingDir == "~" {
			workingDir = home
		} else {
			workingDir = filepath.Join(home, strings.TrimPrefix(workingDir, "~/"))
		}
	}
	absolute, err := filepath.Abs(workingDir)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w: %q is not a directory", ErrInvalid, canonical)
	}
	directory, err := os.Open(canonical)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if err := directory.Close(); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return canonical, nil
}

// Browse lists at most maxDirectories direct child directories.
func Browse(requested, fallback string) (Listing, error) {
	path, err := Resolve(requested, fallback)
	if err != nil {
		return Listing{}, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return Listing{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	listing := Listing{Path: path, Parent: filepath.Dir(path), Directories: make([]Entry, 0)}
	for _, entry := range entries {
		childPath := filepath.Join(path, entry.Name())
		info, infoErr := os.Stat(childPath)
		if infoErr != nil || !info.IsDir() {
			continue
		}
		canonicalChild, resolveErr := filepath.EvalSymlinks(childPath)
		if resolveErr != nil {
			continue
		}
		listing.Directories = append(listing.Directories, Entry{Name: entry.Name(), Path: canonicalChild})
		if len(listing.Directories) == maxDirectories {
			break
		}
	}
	return listing, nil
}
