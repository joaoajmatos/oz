package langs

import "github.com/joaoajmatos/oz/internal/shell/readfilter"

type genericReader struct{}

func (genericReader) Name() string { return "generic" }
func (genericReader) Extensions() []string {
	return nil
}
func (genericReader) Filter(content string, _ readfilter.Options) (string, error) {
	return content, nil
}

func init() {
	readfilter.Register(genericReader{})
}
