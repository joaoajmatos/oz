package bm25

import "math"

// FieldDoc is the generic shape consumed by the BM25 core.
type FieldDoc interface {
	Fields() map[string][]string
}

// BM25Field describes one scored field.
type BM25Field struct {
	Name   string
	Weight float64
	B      float64
}

// NormTF returns the BM25F normalised term frequency for one field.
func NormTF(tf, fieldLen int, avgLen, b float64) float64 {
	if tf == 0 || fieldLen == 0 {
		return 0
	}
	norm := 1 - b + b*float64(fieldLen)/avgLen
	if norm == 0 {
		norm = 1
	}
	return float64(tf) / norm
}

// AvgFieldLengths returns the mean token length of each configured field.
func AvgFieldLengths(docs []FieldDoc, fields []BM25Field) map[string]float64 {
	out := make(map[string]float64, len(fields))
	if len(docs) == 0 {
		for _, f := range fields {
			out[f.Name] = 1
		}
		return out
	}
	for _, f := range fields {
		total := 0.0
		for _, d := range docs {
			total += float64(len(d.Fields()[f.Name]))
		}
		avg := total / float64(len(docs))
		if avg < 1 {
			avg = 1
		}
		out[f.Name] = avg
	}
	return out
}

// ComputeDF returns how many documents contain each term across any field.
func ComputeDF(docs []FieldDoc) map[string]int {
	df := make(map[string]int)
	for _, d := range docs {
		seen := make(map[string]bool)
		for _, tokens := range d.Fields() {
			for _, t := range tokens {
				if !seen[t] {
					df[t]++
					seen[t] = true
				}
			}
		}
	}
	return df
}

// BM25Score computes the BM25F pseudo-TF score for one document.
func BM25Score(
	terms []string,
	docFields map[string][]string,
	fields []BM25Field,
	k1 float64,
	avgLen map[string]float64,
	df map[string]int,
	N int,
) float64 {
	_, total := bm25DocScore(terms, docFields, fields, k1, avgLen, df, N, nil)
	return total
}

// FieldScoreShares partitions the BM25F document score across configured fields.
// For each query term, the term contribution idf*tfTilde/(k1+tfTilde) is split
// in proportion to each field's share of the weighted sum tfTilde, so the
// per-field values sum to the result of [BM25Score] for the same inputs.
// Useful for tests and debugging when a block body duplicates benchmark queries.
func FieldScoreShares(
	terms []string,
	docFields map[string][]string,
	fields []BM25Field,
	k1 float64,
	avgLen map[string]float64,
	df map[string]int,
	N int,
) map[string]float64 {
	acc := make(map[string]float64, len(fields))
	bm25DocScore(terms, docFields, fields, k1, avgLen, df, N, acc)
	return acc
}

// bm25DocScore returns the BM25F total. If acc is non-nil, it adds each field's
// share of the total into acc (per FieldScoreShares).
func bm25DocScore(
	terms []string,
	docFields map[string][]string,
	fields []BM25Field,
	k1 float64,
	avgLen map[string]float64,
	df map[string]int,
	N int,
	acc map[string]float64,
) (map[string]float64, float64) {
	if len(terms) == 0 {
		return acc, 0
	}
	if acc != nil {
		for _, f := range fields {
			acc[f.Name] = 0
		}
	}
	nFloat := float64(N)
	total := 0.0
	parts := make([]float64, len(fields))
	for _, term := range terms {
		dfVal := float64(df[term])
		idf := math.Log((nFloat-dfVal+0.5)/(dfVal+0.5) + 1)
		if idf < 1.0 {
			idf = 1.0
		}
		tfTilde := 0.0
		for i, f := range fields {
			tokens := docFields[f.Name]
			tf := termFreq(term, tokens)
			p := f.Weight * NormTF(tf, len(tokens), avgLen[f.Name], f.B)
			parts[i] = p
			tfTilde += p
		}
		termContrib := idf * tfTilde / (k1 + tfTilde)
		total += termContrib
		if acc == nil || tfTilde == 0 {
			continue
		}
		for i, f := range fields {
			acc[f.Name] += termContrib * (parts[i] / tfTilde)
		}
	}
	return acc, total
}

func termFreq(term string, tokens []string) int {
	tf := 0
	for _, tok := range tokens {
		if tok == term {
			tf++
		}
	}
	return tf
}

