package workspace

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/joaoajmatos/oz/internal/convention"
)

// Workspace represents a detected oz workspace on disk.
type Workspace struct {
	// Root is the resolved workspace root directory (absolute path).
	Root string
}

// New discovers the workspace root starting from path.
//
// It resolves path to an absolute directory, then walks upward parent by parent
// until it finds a directory that contains every required root file (see
// convention.RootFiles). That directory becomes the Root field. This lets CLI
// commands run from any subdirectory inside an oz workspace when path is "." or
// any path inside the tree.
//
// If no ancestor has the required files, Root is the absolute starting path and
// Valid usually returns false.
func New(path string) (*Workspace, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return &Workspace{Root: detectWorkspaceRoot(abs)}, nil
}

func detectWorkspaceRoot(path string) string {
	current := path
	for {
		if hasRequiredRootFiles(current) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return path
		}
		current = parent
	}
}

func hasRequiredRootFiles(path string) bool {
	for file, status := range convention.RootFiles {
		if status != "required" {
			continue
		}
		if _, err := os.Stat(filepath.Join(path, file)); err != nil {
			return false
		}
	}
	return true
}

// Valid reports whether the workspace has all required root files.
func (w *Workspace) Valid() bool {
	for file, status := range convention.RootFiles {
		if status == "required" {
			if _, err := os.Stat(filepath.Join(w.Root, file)); err != nil {
				return false
			}
		}
	}
	return true
}

// Manifest holds the key fields parsed from OZ.md.
type Manifest struct {
	Name        string
	Description string
}

// ReadManifest parses the project name and description from OZ.md.
func (w *Workspace) ReadManifest() (Manifest, error) {
	f, err := os.Open(filepath.Join(w.Root, "OZ.md"))
	if err != nil {
		return Manifest{}, err
	}
	defer f.Close()

	var m Manifest
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if v, ok := parseField(line, "project"); ok {
			m.Name = v
		} else if v, ok := parseField(line, "description"); ok {
			m.Description = v
		}
	}
	return m, scanner.Err()
}

// parseField extracts the value from a "key: value" line in OZ.md.
func parseField(line, key string) (string, bool) {
	prefix := key + ": "
	if strings.HasPrefix(line, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(line, prefix)), true
	}
	return "", false
}

// Agents returns the names of all registered agents (subdirs of agents/).
func (w *Workspace) Agents() ([]string, error) {
	agentsDir := filepath.Join(w.Root, "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil, err
	}
	var agents []string
	for _, e := range entries {
		if e.IsDir() {
			agents = append(agents, e.Name())
		}
	}
	return agents, nil
}

// Layer represents one level of the source of truth hierarchy.
type Layer struct {
	Name   string
	Path   string
	Exists bool
}

// HierarchyLayers returns all hierarchy layers with their existence status.
func (w *Workspace) HierarchyLayers() []Layer {
	layers := make([]Layer, 0, len(convention.SourceOfTruthOrder))
	for _, name := range convention.SourceOfTruthOrder {
		path := filepath.Join(w.Root, string(name))
		_, err := os.Stat(path)
		layers = append(layers, Layer{
			Name:   string(name),
			Path:   path,
			Exists: err == nil,
		})
	}
	return layers
}
