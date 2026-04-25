package run

import "github.com/joaoajmatos/oz/internal/shell/envelope"

type Result struct {
	Envelope envelope.RunResult
	ExitCode int
}
