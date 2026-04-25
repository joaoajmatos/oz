package langs

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"

	"github.com/joaoajmatos/oz/internal/shell/readfilter"
)

const bodyPreviewLines = 4

var (
	goBlockCommentsRE = regexp.MustCompile(`(?s)/\*.*?\*/`)
	goLineCommentRE   = regexp.MustCompile(`//.*$`)
)

type golangReader struct{}

func (golangReader) Name() string        { return "go" }
func (golangReader) Extensions() []string { return []string{".go"} }

func (golangReader) Filter(content string, opts readfilter.Options) (string, error) {
	out, err := structuralFilterGo(content, opts.UltraCompact)
	if err != nil {
		return fallbackGoFilter(content), nil
	}
	return out, nil
}

func structuralFilterGo(content string, ultra bool) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", content, 0)
	if err != nil {
		return "", err
	}

	lines := strings.Split(content, "\n")
	lineAt := func(n int) string {
		if n < 1 || n > len(lines) {
			return ""
		}
		return lines[n-1]
	}

	var out []string
	out = append(out, lineAt(fset.Position(f.Package).Line))

	for _, decl := range f.Decls {
		out = append(out, "")
		start := fset.Position(decl.Pos()).Line
		end := fset.Position(decl.End()).Line

		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Body == nil {
				for ln := start; ln <= end; ln++ {
					out = append(out, stripInlineComment(lineAt(ln)))
				}
				continue
			}
			openBrace := fset.Position(d.Body.Lbrace).Line
			closeBrace := fset.Position(d.Body.Rbrace).Line
			if openBrace == closeBrace {
				for ln := start; ln <= end; ln++ {
					out = append(out, stripInlineComment(lineAt(ln)))
				}
				continue
			}
			for ln := start; ln <= openBrace; ln++ {
				out = append(out, stripInlineComment(lineAt(ln)))
			}
			bodyLines := closeBrace - openBrace - 1
			if ultra {
				out = append(out, "\t...")
			} else {
				preview := min(bodyPreviewLines, bodyLines)
				for ln := openBrace + 1; ln <= openBrace+preview; ln++ {
					out = append(out, stripInlineComment(lineAt(ln)))
				}
				if bodyLines > preview {
					out = append(out, fmt.Sprintf("\t... (%d lines omitted)", bodyLines-preview))
				}
			}
			out = append(out, lineAt(closeBrace))

		case *ast.GenDecl:
			for ln := start; ln <= end; ln++ {
				out = append(out, stripInlineComment(lineAt(ln)))
			}
		}
	}

	return collapseAndTrim(out), nil
}

func stripInlineComment(line string) string {
	if strings.HasPrefix(strings.TrimSpace(line), "//") {
		return ""
	}
	return strings.TrimRight(goLineCommentRE.ReplaceAllString(line, ""), " \t")
}

func collapseAndTrim(lines []string) string {
	out := make([]string, 0, len(lines))
	blanks := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blanks++
			if blanks > 1 {
				continue
			}
			out = append(out, "")
		} else {
			blanks = 0
			out = append(out, line)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n")) + "\n"
}

func fallbackGoFilter(content string) string {
	content = goBlockCommentsRE.ReplaceAllString(content, "")
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	blankCount := 0
	for _, line := range lines {
		line = goLineCommentRE.ReplaceAllString(line, "")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blankCount++
			if blankCount > 1 {
				continue
			}
			out = append(out, "")
			continue
		}
		blankCount = 0
		out = append(out, strings.TrimRight(line, " \t"))
	}
	return strings.TrimSpace(strings.Join(out, "\n")) + "\n"
}

func init() {
	readfilter.Register(golangReader{})
}
