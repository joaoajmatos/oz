package readfilter

type LanguageReader interface {
	Name() string
	Extensions() []string
	Filter(content string, opts Options) (string, error)
}

type Options struct {
	Path         string
	Stdin        bool
	Content      string
	MaxLines     *int
	TailLines    *int
	LineNumbers  bool
	NoFilter     bool
	UltraCompact bool
}

type Result struct {
	Path           string
	Language       string
	Content        string
	Warnings       []string
	TokenEstBefore int
	TokenEstAfter  int
}
