package workspace

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/oz-tools/oz/internal/convention"
)

// Workspace represents a detected oz workspace on disk.
type Workspace struct {
	Root string
}

// New loads a workspace from the given path.
func New(path string) (*Workspace, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return &Workspace{Root: abs}, nil
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
	layers := make([]Layer, 0, len(convention.Hierarchy))
	for _, name := range convention.Hierarchy {
		path := filepath.Join(w.Root, name)
		_, err := os.Stat(path)
		layers = append(layers, Layer{
			Name:   name,
			Path:   path,
			Exists: err == nil,
		})
	}
	return layers
}
