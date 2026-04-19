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
	tokens := TokenizeMulti("test testing tested")
	if len(tokens) < 2 {
		t.Errorf("expected multiple tokens (duplicates allowed), got %v", tokens)
	}
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
