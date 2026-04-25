package filter

// init registers filters in match-precedence order (first Match wins).
func init() {
	for _, f := range orderedFilters() {
		Register(f)
	}
}

func orderedFilters() []Filter {
	return []Filter{
		&gitStatusFilter{},
		&gitDiffFilter{},
		&gitLogFilter{},
		&gitBlameFilter{},
		&gitShowFilter{},
		&goTestFilter{},
		&goBuildFilter{},
		&goVetFilter{},
		&staticcheckFilter{},
		&rgFilter{},
		&dockerFilter{},
		&httpFilter{},
		&jsonFilter{},
		&pytestFilter{},
		&npmFilter{},
		&findFilter{},
		&treeFilter{},
		&lsFilter{},
		&makeFilter{},
		&cargoFilter{},
		&diffFilter{},
		&envFilter{},
		&wcFilter{},
		&dfFilter{},
		&psFilter{},
		&topBatchFilter{},
	}
}
