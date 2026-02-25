// pkg/manifest/loader.go
package manifest

import (
	"io/fs"
	"os"
)

// DiskFS wraps os.DirFS to satisfy fs.FS for loading manifests
// from a real filesystem path (useful in development/debugging).
func DiskFS(root string) fs.FS {
	return os.DirFS(root)
}

// SubFS returns a sub-filesystem rooted at dir within the given fs.FS.
// Useful for scoping an embed.FS to a particular testdata sub-directory.
func SubFS(fsys fs.FS, dir string) (fs.FS, error) {
	return fs.Sub(fsys, dir)
}

// ListManifests returns all YAML file paths within a directory of an fs.FS,
// sorted deterministically.
func ListManifests(fsys fs.FS, dir string) ([]string, error) {
	var all []string

	yamlFiles, err := fs.Glob(fsys, dir+"/*.yaml")
	if err != nil {
		return nil, err
	}
	all = append(all, yamlFiles...)

	ymlFiles, _ := fs.Glob(fsys, dir+"/*.yml")
	all = append(all, ymlFiles...)

	return all, nil
}
