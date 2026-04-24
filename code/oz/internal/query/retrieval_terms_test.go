package query

import (
	"reflect"
	"testing"
)

func TestDiscriminativeRetrievalTerms_DropsCodeWhenOtherStemsPresent(t *testing.T) {
	got := DiscriminativeRetrievalTerms([]string{"code", "index", "graph"})
	want := []string{"index", "graph"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestDiscriminativeRetrievalTerms_SingleTermUnchanged(t *testing.T) {
	if g := DiscriminativeRetrievalTerms([]string{"code"}); !reflect.DeepEqual(g, []string{"code"}) {
		t.Fatalf("got %v, want [code]", g)
	}
}

func TestSemanticRetrievalQueryTerms_AddsCodeindexWhenCodeAndIndex(t *testing.T) {
	// "code" + "index" should still match path token "codeindex" in ADR names.
	terms := Tokenize("code indexing")
	got := SemanticRetrievalQueryTerms(terms)
	if len(got) < 2 {
		t.Fatalf("expected at least 2 terms, got %v", got)
	}
	var hasIndex, hasCompound bool
	for _, t := range got {
		if t == "index" {
			hasIndex = true
		}
		if t == Stem("codeindex") {
			hasCompound = true
		}
	}
	if !hasIndex {
		t.Fatalf("expected index in %v", got)
	}
	if !hasCompound {
		t.Fatalf("expected codeindex stem %q in %v", Stem("codeindex"), got)
	}
}

func TestDiscriminativeRetrievalTerms_AllNoiseFallsBack(t *testing.T) {
	// If every term is noise, the original slice is returned unchanged.
	got := DiscriminativeRetrievalTerms([]string{"code", "code"})
	if len(got) != 2 || got[0] != "code" || got[1] != "code" {
		t.Fatalf("expected [code code], got %v", got)
	}
}
