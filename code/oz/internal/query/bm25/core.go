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
	if len(terms) == 0 {
		return 0
	}
	score := 0.0
	nFloat := float64(N)
	for _, term := range terms {
		dfVal := float64(df[term])
		idf := math.Log((nFloat-dfVal+0.5)/(dfVal+0.5) + 1)
		if idf < 1.0 {
			idf = 1.0
		}

		tfTilde := 0.0
		for _, f := range fields {
			tokens := docFields[f.Name]
			tf := termFreq(term, tokens)
			tfTilde += f.Weight * NormTF(tf, len(tokens), avgLen[f.Name], f.B)
		}

		score += idf * tfTilde / (k1 + tfTilde)
	}
	return score
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

