package query

// DiscriminativeRetrievalTerms returns query stems for **semantic** retrieval
// (relevant_concepts, implementing_packages, and code entry points) with
// ultra-common stems removed when the query has at least one more specific stem.
//
// [SemanticRetrievalQueryTerms] builds on this by adding a `codeindex` stem when
// the query had both "code" and "index" so file paths and packages match.
//
// The stem "code" (and a small set of other workspace-generic tokens) appears in
// nearly every reviewed concept description; without this filter, almost every
// concept scores above the relevance floor and generic queries surface an
// arbitrary top-K set that looks wrong (e.g. "how is code indexing done"
// favoring any concept that mentions "code" over "index" specifically).
//
// If removing noise would leave no terms, the original slice is returned so
// single-term queries like "code" or "index" still behave.
func DiscriminativeRetrievalTerms(terms []string) []string {
	if len(terms) <= 1 {
		return terms
	}
	var out []string
	for _, t := range terms {
		if !retrievalNoiseStems[t] {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return terms
	}
	return out
}

// SemanticRetrievalQueryTerms is like [DiscriminativeRetrievalTerms] but appends
// a compound when the original query contained the noise stem "code" and the
// result still has "index" (e.g. "code indexing" → "code" + "index" after
// stemming). File paths and import segments often use a **single** token
// "codeindex" (Porter usually keeps that stem), so BM25 with only "index" does
// not match `...-codeindex.md` or `code/.../index/`. Appending
// [Stem]("codeindex") keeps path/symbol matches for relevant_concepts,
// implementing_packages, and code entry points.
func SemanticRetrievalQueryTerms(queryTerms []string) []string {
	out := DiscriminativeRetrievalTerms(queryTerms)
	if len(out) == 0 {
		return out
	}
	var hadCode bool
	for _, t := range queryTerms {
		if t == "code" {
			hadCode = true
			break
		}
	}
	if !hadCode {
		return out
	}
	var hasIndex bool
	for _, t := range out {
		if t == "index" {
			hasIndex = true
			break
		}
	}
	if !hasIndex {
		return out
	}
	compound := Stem("codeindex")
	if compound == "" || stringInStems(out, compound) {
		return out
	}
	return append(out, compound)
}

func stringInStems(stems []string, s string) bool {
	for _, t := range stems {
		if t == s {
			return true
		}
	}
	return false
}

// Stems to drop when (len(terms) > 1) for concept + code-symbol package scoring.
// Keep the set tiny and obvious — prefer tuning retrieval.toml floors for edge cases.
var retrievalNoiseStems = map[string]bool{
	"code": true,
}
