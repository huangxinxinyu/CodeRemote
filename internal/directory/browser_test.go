package directory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrowseReturnsCanonicalParentAndSortedChildDirectories(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, name := range []string{"zeta", "alpha"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	listing, err := Browse(root, "")
	if err != nil {
		t.Fatal(err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if listing.Path != canonicalRoot || listing.Parent != filepath.Dir(canonicalRoot) {
		t.Fatalf("listing path/parent = %q / %q, want %q / %q", listing.Path, listing.Parent, canonicalRoot, filepath.Dir(canonicalRoot))
	}
	if len(listing.Directories) != 2 || listing.Directories[0].Name != "alpha" || listing.Directories[1].Name != "zeta" {
		t.Fatalf("directories = %#v, want sorted directories only", listing.Directories)
	}
	if listing.Directories[0].Path != filepath.Join(canonicalRoot, "alpha") {
		t.Fatalf("first directory path = %q, want canonical child path", listing.Directories[0].Path)
	}
}

func TestResolveExpandsHomeAndRejectsFiles(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := Resolve("~", "")
	if err != nil {
		t.Fatal(err)
	}
	canonicalHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != canonicalHome {
		t.Fatalf("Resolve(~) = %q, want %q", resolved, canonicalHome)
	}

	filePath := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(filePath, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(filePath, ""); err == nil {
		t.Fatal("Resolve(file) succeeded, want an error")
	}
}
