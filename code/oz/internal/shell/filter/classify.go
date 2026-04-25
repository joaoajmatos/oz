package filter

type ID string

const (
	FilterNone      ID = "none"
	FilterGeneric   ID = "generic"
	FilterGitStatus ID = "git.status"
	FilterGitDiff   ID = "git.diff"
	FilterGitLog    ID = "git.log"
	FilterGitBlame  ID = "git.blame"
	FilterGitShow   ID = "git.show"
	FilterRG        ID = "rg"
	FilterGoTest    ID = "go.test"
	FilterGoBuild   ID = "go.build"
	FilterGoVet     ID = "go.vet"
	FilterLs        ID = "ls"
	FilterFind      ID = "find"
	FilterTree      ID = "tree"
	FilterJSON      ID = "json"
	FilterMake      ID = "make"
	FilterNpm       ID = "npm"
	FilterDocker    ID = "docker"
	FilterHTTP      ID = "http"
	FilterEnv       ID = "env"
	FilterWc        ID = "wc"
	FilterDiff      ID = "diff"
	FilterPs        ID = "ps"
	FilterDf        ID = "df"
	FilterCargo     ID = "cargo"
	FilterPytest    ID = "pytest"
)

func Classify(args []string) ID {
	f := lookup(args)
	if f == nil {
		return FilterGeneric
	}
	return f.ID()
}
