package query

import (
	"strings"
	"unicode"
)

// Tokenize runs the query tokenization pipeline with unigrams only (no bigrams).
// Equivalent to TokenizeQuery(text, false).
func Tokenize(text string) []string {
	return TokenizeQuery(text, false)
}

// TokenizeQuery tokenizes query text: lowercase → split on non-letters →
// filter stopwords → stem → deduplicated unigrams in first-seen order.
// When useBigrams is true, also appends deduplicated adjacent stem bigrams
// as "stem_i_stem_j" in order after the last new unigram would appear.
func TokenizeQuery(text string, useBigrams bool) []string {
	seq := stemSequence(text)
	if len(seq) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(seq)*2)
	var out []string
	for _, s := range seq {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	if useBigrams {
		for i := 0; i < len(seq)-1; i++ {
			bg := seq[i] + "_" + seq[i+1]
			if len(bg) < 3 {
				continue
			}
			if !seen[bg] {
				seen[bg] = true
				out = append(out, bg)
			}
		}
	}
	return out
}

// stemSequence returns stemmed tokens in word order (duplicates allowed).
func stemSequence(text string) []string {
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

// TokenizeMulti tokenizes text for document fields, allowing duplicate stems
// for term-frequency counting. When useBigrams is true, appends one token per
// adjacent stem pair after all unigrams.
func TokenizeMulti(text string, useBigrams bool) []string {
	tokens := split(strings.ToLower(text))
	var uni []string
	for _, tok := range tokens {
		if len(tok) < 2 || stopwords[tok] {
			continue
		}
		s := Stem(tok)
		if len(s) >= 2 {
			uni = append(uni, s)
		}
	}
	if !useBigrams || len(uni) < 2 {
		return uni
	}
	out := make([]string, len(uni), len(uni)+len(uni)-1)
	copy(out, uni)
	for i := 0; i < len(uni)-1; i++ {
		out = append(out, uni[i]+"_"+uni[i+1])
	}
	return out
}

// TokenizePaths tokenizes path globs into deduplicated stemmed unigrams (no bigrams).
func TokenizePaths(paths []string) []string {
	return TokenizePathsQuery(paths, false)
}

// TokenizePathsQuery is like TokenizePaths but optionally adds path bigrams.
func TokenizePathsQuery(paths []string, useBigrams bool) []string {
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
	return TokenizeQuery(strings.Join(parts, " "), useBigrams)
}

// TokenizePathsMulti is the multi-occurrence version of TokenizePaths for BM25F fields.
func TokenizePathsMulti(paths []string, useBigrams bool) []string {
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
	return TokenizeMulti(strings.Join(parts, " "), useBigrams)
}

// split breaks s into tokens on any non-alphabetic character.
func split(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
}
