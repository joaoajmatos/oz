package run

type Mode string

const (
	ModeCompact Mode = "compact"
	ModeRaw     Mode = "raw"
)

type TeeMode string

const (
	TeeModeFailures TeeMode = "failures"
	TeeModeAlways   TeeMode = "always"
	TeeModeNever    TeeMode = "never"
)

type Options struct {
	Mode         Mode
	TeeMode      TeeMode
	NoTrack      bool
	UltraCompact bool
}
