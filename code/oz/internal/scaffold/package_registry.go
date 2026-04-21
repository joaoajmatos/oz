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

// ValidPackageIDs returns sorted package IDs for help text and errors.
func ValidPackageIDs() []string {
	ids := make([]string, 0, len(packageInstallers))
	for id := range packageInstallers {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// InstallPackage installs an optional package by ID into an existing workspace root.
// It returns the list of paths created or updated (relative to root). On unknown ID,
// returns [*ErrUnknownPackage].
func InstallPackage(id, root string, force bool) ([]string, error) {
	fn, ok := packageInstallers[id]
	if !ok {
		return nil, &ErrUnknownPackage{ID: id}
	}
	return fn(root, force)
}

// packageInstallers maps package ID → installer. Add new packages here only.
var packageInstallers = map[string]func(root string, force bool) ([]string, error){
	"pm": installPMPackage,
}
