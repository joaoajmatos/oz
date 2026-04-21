package scaffold

import (
	"fmt"
	"slices"
	"strings"
)

// ErrUnknownPackage is returned when InstallPackage is called with an ID
// that is not in the registry.
type ErrUnknownPackage struct {
	ID string
}

func (e *ErrUnknownPackage) Error() string {
	return fmt.Sprintf("unknown package %q — valid packages: %s", e.ID, strings.Join(ValidPackageIDs(), ", "))
}

// PackageCatalogEntry describes one optional package for help and `oz add list`.
type PackageCatalogEntry struct {
	ID            string
	Summary       string
	SupportsForce bool // whether --force is accepted for this package
}

// PackageCatalog returns package metadata in stable ID order for display.
func PackageCatalog() []PackageCatalogEntry {
	ids := ValidPackageIDs()
	out := make([]PackageCatalogEntry, 0, len(ids))
	for _, id := range ids {
		def := packageDefs[id]
		out = append(out, PackageCatalogEntry{
			ID:            id,
			Summary:       def.summary,
			SupportsForce: def.supportsForce,
		})
	}
	return out
}

// ValidPackageIDs returns sorted package IDs for help text and errors.
func ValidPackageIDs() []string {
	ids := make([]string, 0, len(packageDefs))
	for id := range packageDefs {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// InstallPackage installs an optional package by ID into an existing workspace root.
// It returns the list of paths created or updated (relative to root). On unknown ID,
// returns [*ErrUnknownPackage].
func InstallPackage(id, root string, force bool) ([]string, error) {
	def, ok := packageDefs[id]
	if !ok {
		return nil, &ErrUnknownPackage{ID: id}
	}
	return def.install(root, force)
}

type packageDef struct {
	install       func(root string, force bool) ([]string, error)
	summary       string
	supportsForce bool
}

// packageDefs is the single registry for optional packages (install + list metadata).
var packageDefs = map[string]packageDef{
	"pm": {
		install:       installPMPackage,
		summary:       "PM agent + skills — PRDs, pre-mortems, stories, sprint rituals",
		supportsForce: true,
	},
}
