package query

import (
	"testing"
)

func TestTokenize_Basic(t *testing.T) {
	tokens := Tokenize("implement the REST API endpoint")
	// "the" is stopword, short words filtered, rest stemmed
	checkContains(t, tokens, "implement")
	checkContains(t, tokens, "rest")
	checkContains(t, tokens, "api")
	checkContains(t, tokens, "endpoint")
	checkAbsent(t, tokens, "the")
}

func TestTokenize_Deduplicates(t *testing.T) {
	tokens := Tokenize("fix fix fixing")
	if len(tokens) != 1 {
		t.Errorf("expected 1 unique token, got %d: %v", len(tokens), tokens)
	}
}

func TestTokenize_Stopwords(t *testing.T) {
	tokens := Tokenize("a the and or but in on at to for of with by from")
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens after stopword removal, got %v", tokens)
	}
}

func TestTokenizePaths_ExtractsComponents(t *testing.T) {
	tokens := TokenizePaths([]string{"code/api/**", "internal/auth/**"})
	checkContains(t, tokens, "code")
	checkContains(t, tokens, "api")
	checkContains(t, tokens, "intern")
	checkContains(t, tokens, "auth")
	// wildcards stripped
	checkAbsent(t, tokens, "*")
}

func TestTokenizeMulti_AllowsDuplicates(t *testing.T) {
	tokens := TokenizeMulti("test testing tested", false)
	if len(tokens) < 2 {
		t.Errorf("expected multiple tokens (duplicates allowed), got %v", tokens)
	}
}

func TestTokenizeQuery_BigramsOptional(t *testing.T) {
	without := TokenizeQuery("implement rest api", false)
	for _, tok := range without {
		if tok == "rest_api" {
			t.Errorf("did not expect bigram token in unigram-only mode, got %v", without)
		}
	}
	with := TokenizeQuery("implement rest api", true)
	checkContains(t, with, "implement")
	checkContains(t, with, "rest")
	checkContains(t, with, "api")
	checkContains(t, with, "rest_api")
}

func TestTokenizeMulti_WithBigrams(t *testing.T) {
	tokens := TokenizeMulti("rest api server", true)
	checkContains(t, tokens, "rest")
	checkContains(t, tokens, "api")
	checkContains(t, tokens, "server")
	checkContains(t, tokens, "rest_api")
	checkContains(t, tokens, "api_server")
}

func checkContains(t *testing.T, tokens []string, want string) {
	t.Helper()
	for _, tok := range tokens {
		if tok == want {
			return
		}
	}
	t.Errorf("expected token %q in %v", want, tokens)
}

func checkAbsent(t *testing.T, tokens []string, notWant string) {
	t.Helper()
	for _, tok := range tokens {
		if tok == notWant {
			t.Errorf("unexpected token %q in %v", notWant, tokens)
			return
		}
	}
}
