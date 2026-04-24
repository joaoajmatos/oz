package bm25

import "testing"

// mapDoc is a test FieldDoc implementation that wraps a plain map.
type mapDoc map[string][]string

func (m mapDoc) Fields() map[string][]string { return m }

func TestNormTF_ZeroInputs(t *testing.T) {
	if got := NormTF(0, 10, 5, 0.5); got != 0 {
		t.Errorf("NormTF(tf=0) = %f, want 0", got)
	}
	if got := NormTF(5, 0, 5, 0.5); got != 0 {
		t.Errorf("NormTF(fieldLen=0) = %f, want 0", got)
	}
}

func TestNormTF_BZeroIsFlat(t *testing.T) {
	// With b=0, length normalisation drops out: result is just tf.
	if got := NormTF(3, 100, 10, 0); got != 3 {
		t.Errorf("NormTF(b=0) = %f, want 3", got)
	}
}

func TestNormTF_BOneNormalisesByLength(t *testing.T) {
	// b=1, fieldLen=avgLen => norm=1 => tf unchanged.
	if got := NormTF(4, 10, 10, 1); got != 4 {
		t.Errorf("NormTF(fieldLen=avgLen, b=1) = %f, want 4", got)
	}
	// Longer-than-average field penalised.
	if got := NormTF(4, 20, 10, 1); got >= 4 {
		t.Errorf("NormTF(fieldLen>avgLen, b=1) = %f, want < 4", got)
	}
}

func TestAvgFieldLengths_EmptyCorpus(t *testing.T) {
	fields := []BM25Field{{Name: "a"}, {Name: "b"}}
	avg := AvgFieldLengths(nil, fields)
	for _, f := range fields {
		if avg[f.Name] != 1 {
			t.Errorf("AvgFieldLengths empty corpus: field %q = %f, want 1 (floor)", f.Name, avg[f.Name])
		}
	}
}

func TestAvgFieldLengths_MeansAndFloor(t *testing.T) {
	fields := []BM25Field{{Name: "title"}, {Name: "body"}}
	docs := []FieldDoc{
		mapDoc{"title": {"a", "b"}, "body": {"x"}},
		mapDoc{"title": {"c"}, "body": nil},
	}
	avg := AvgFieldLengths(docs, fields)
	if avg["title"] != 1.5 {
		t.Errorf("avg[title] = %f, want 1.5", avg["title"])
	}
	// body totals 1 token across 2 docs = 0.5 -> floors to 1.
	if avg["body"] != 1 {
		t.Errorf("avg[body] = %f, want 1 (floor)", avg["body"])
	}
}

func TestComputeDF_CountsDocNotField(t *testing.T) {
	// Term "x" appears in two fields of doc1 but must count once.
	docs := []FieldDoc{
		mapDoc{"a": {"x", "y"}, "b": {"x"}},
		mapDoc{"a": {"y"}, "b": nil},
	}
	df := ComputeDF(docs)
	if df["x"] != 1 {
		t.Errorf("df[x] = %d, want 1 (same doc, multiple fields)", df["x"])
	}
	if df["y"] != 2 {
		t.Errorf("df[y] = %d, want 2", df["y"])
	}
	if _, ok := df["z"]; ok {
		t.Errorf("df[z] present, want absent")
	}
}

func TestBM25Score_EmptyTerms(t *testing.T) {
	fields := []BM25Field{{Name: "title", Weight: 1, B: 0.5}}
	got := BM25Score(nil, map[string][]string{"title": {"a"}}, fields, 1.2, map[string]float64{"title": 1}, nil, 1)
	if got != 0 {
		t.Errorf("BM25Score(no terms) = %f, want 0", got)
	}
}

func TestBM25Score_HigherOnMatchingDoc(t *testing.T) {
	fields := []BM25Field{
		{Name: "title", Weight: 2.0, B: 0.5},
		{Name: "body", Weight: 1.0, B: 0.75},
	}
	docs := []FieldDoc{
		mapDoc{"title": {"rest", "api"}, "body": {"implement", "endpoint"}},
		mapDoc{"title": {"ui"}, "body": {"react", "component"}},
	}
	avg := AvgFieldLengths(docs, fields)
	df := ComputeDF(docs)
	terms := []string{"rest", "api"}

	hit := BM25Score(terms, docs[0].Fields(), fields, 1.2, avg, df, len(docs))
	miss := BM25Score(terms, docs[1].Fields(), fields, 1.2, avg, df, len(docs))

	if hit <= miss {
		t.Errorf("BM25Score matching doc (%f) should exceed non-matching (%f)", hit, miss)
	}
}

func TestBM25Score_ZeroWhenNoFieldMatch(t *testing.T) {
	fields := []BM25Field{{Name: "title", Weight: 1, B: 0.5}}
	doc := map[string][]string{"title": {"alpha"}}
	avg := map[string]float64{"title": 1}
	df := map[string]int{"alpha": 1}
	got := BM25Score([]string{"zeta"}, doc, fields, 1.2, avg, df, 1)
	if got != 0 {
		t.Errorf("BM25Score(no match) = %f, want 0", got)
	}
}

