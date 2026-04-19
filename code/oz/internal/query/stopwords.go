package query

// stopwords is the list of English function words filtered before scoring.
// Only include words that carry no discriminative signal in workspace queries.
var stopwords = map[string]bool{
	"a": true, "an": true, "the": true,
	"and": true, "or": true, "but": true, "nor": true,
	"in": true, "on": true, "at": true, "to": true, "for": true,
	"of": true, "with": true, "by": true, "from": true, "into": true,
	"up": true, "out": true, "as": true, "is": true, "it": true,
	"its": true, "be": true, "been": true, "was": true, "were": true,
	"are": true, "am": true, "will": true, "would": true, "could": true,
	"should": true, "may": true, "might": true, "can": true,
	"do": true, "does": true, "did": true, "done": true, "have": true,
	"has": true, "had": true, "this": true, "that": true, "these": true,
	"those": true, "i": true, "we": true, "you": true, "he": true,
	"she": true, "they": true, "me": true, "us": true, "him": true,
	"her": true, "them": true, "my": true, "our": true, "your": true,
	"his": true, "their": true, "what": true, "which": true, "who": true,
	"how": true, "when": true, "where": true, "why": true, "all": true,
	"each": true, "every": true, "any": true, "some": true, "no": true,
	"not": true, "so": true, "if": true, "then": true, "also": true,
	"just": true, "after": true, "before": true, "than": true, "about": true,
	"more": true, "over": true, "under": true, "same": true,
	"such": true, "between": true, "there": true, "here": true,
	"via": true, "per": true, "vs": true,
}
