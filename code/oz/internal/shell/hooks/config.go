package hooks

type Mode string

const (
	ModeSuggest Mode = "suggest"
	ModeRewrite Mode = "rewrite"
)

type Config struct {
	Enabled         bool
	Mode            Mode
	ExcludeCommands []string
}

func DefaultConfig() Config {
	return Config{
		Enabled:         true,
		Mode:            ModeSuggest,
		ExcludeCommands: nil,
	}
}
