package query

import (
	"strings"
	"unicode"
)

// Tokenize runs the full tokenization pipeline on text:
// lowercase → split on non-alpha → filter stopwords → stem.
// Returns a deduplicated slice of stemmed tokens.
func Tokenize(text string) []string {
	tokens := split(strings.ToLower(text))
	var out []string
	seen := make(map[string]bool, len(tokens))
	for _, tok := range tokens {
		if len(tok) < 2 || stopwords[tok] {
			continue
		}
		s := Stem(tok)
		if len(s) < 2 {
			continue
		}
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// TokenizeMulti tokenizes text but allows duplicate stems (for TF counting).
func TokenizeMulti(text string) []string {
	tokens := split(strings.ToLower(text))
	var out []string
	for _, tok := range tokens {
		if len(tok) < 2 || stopwords[tok] {
			continue
		}
		s := Stem(tok)
		if len(s) >= 2 {
			out = append(out, s)
		}
	}
	return out
}

// TokenizePaths tokenizes a slice of file/glob paths into stemmed tokens.
// Each path is split on '/', '-', '_', '.', and '*' boundaries.
func TokenizePaths(paths []string) []string {
	var parts []string
	for _, p := range paths {
		// Replace path separators and wildcards with spaces.
		clean := strings.Map(func(r rune) rune {
			if r == '/' || r == '-' || r == '_' || r == '.' || r == '*' {
				return ' '
			}
			return r
		}, p)
		parts = append(parts, clean)
	}
	return Tokenize(strings.Join(parts, " "))
}

// TokenizePathsMulti is the multi-occurrence version of TokenizePaths.
func TokenizePathsMulti(paths []string) []string {
	var parts []string
	for _, p := range paths {
		clean := strings.Map(func(r rune) rune {
			if r == '/' || r == '-' || r == '_' || r == '.' || r == '*' {
				return ' '
			}
			return r
		}, p)
		parts = append(parts, clean)
	}
	return TokenizeMulti(strings.Join(parts, " "))
}

// split breaks s into tokens on any non-alphabetic character.
func split(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
}
