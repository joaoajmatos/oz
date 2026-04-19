package query

// Stem returns the Porter-stemmed form of word (lowercase input expected).
// Reference: M.F. Porter, "An algorithm for suffix stripping",
// Program 14(3) pp 130-137, 1980.
func Stem(word string) string {
	if len(word) <= 2 {
		return word
	}

	w := []byte(word)
	w = step1a(w)
	w = step1b(w)
	w = step1c(w)
	w = step2(w)
	w = step3(w)
	w = step4(w)
	w = step5a(w)
	w = step5b(w)
	return string(w)
}

// ---- helpers ----------------------------------------------------------------

// isVowelAt reports whether the character at index i in w is a vowel.
// y is treated as a vowel when preceded by a consonant.
func isVowelAt(w []byte, i int) bool {
	switch w[i] {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	case 'y':
		return i > 0 && !isVowelAt(w, i-1)
	}
	return false
}

// measure returns the number of VC (vowel-run / consonant-run) sequences in w.
// It counts transitions from a vowel run to a consonant run.
func measure(w []byte) int {
	m := 0
	inVowel := false
	for i := range w {
		if isVowelAt(w, i) {
			inVowel = true
		} else if inVowel {
			m++
			inVowel = false
		}
	}
	return m
}

// containsVowel reports whether w contains at least one vowel.
func containsVowel(w []byte) bool {
	for i := range w {
		if isVowelAt(w, i) {
			return true
		}
	}
	return false
}

// endsDoubleC reports whether w ends in a double consonant.
func endsDoubleC(w []byte) bool {
	n := len(w)
	return n >= 2 && w[n-1] == w[n-2] && !isVowelAt(w, n-1)
}

// endsCVC reports whether w ends in consonant-vowel-consonant where the
// final consonant is not W, X, or Y.
func endsCVC(w []byte) bool {
	n := len(w)
	if n < 3 {
		return false
	}
	last := w[n-1]
	if last == 'w' || last == 'x' || last == 'y' {
		return false
	}
	return !isVowelAt(w, n-1) && isVowelAt(w, n-2) && !isVowelAt(w, n-3)
}

// hasSuffix is a convenience wrapper.
func hasSuffix(w []byte, s string) bool {
	n, sn := len(w), len(s)
	if sn > n {
		return false
	}
	return string(w[n-sn:]) == s
}

// stem returns w with the last sn bytes replaced by repl.
func replace(w []byte, sn int, repl string) []byte {
	return append(w[:len(w)-sn], []byte(repl)...)
}

// ---- steps ------------------------------------------------------------------

func step1a(w []byte) []byte {
	switch {
	case hasSuffix(w, "sses"):
		return replace(w, 4, "ss")
	case hasSuffix(w, "ies"):
		return replace(w, 3, "i")
	case hasSuffix(w, "ss"):
		// no change
	case hasSuffix(w, "s"):
		return replace(w, 1, "")
	}
	return w
}

func step1b(w []byte) []byte {
	// Rule priority: eed is tried first (most specific).
	if hasSuffix(w, "eed") {
		stem := w[:len(w)-3]
		if measure(stem) > 0 {
			return replace(w, 1, "") // eed → ee
		}
		// eed matched but condition failed — do NOT fall through to ed.
		return w
	}

	var fired bool
	switch {
	case hasSuffix(w, "ed"):
		stem := w[:len(w)-2]
		if containsVowel(stem) {
			w = replace(w, 2, "")
			fired = true
		}
	case hasSuffix(w, "ing"):
		stem := w[:len(w)-3]
		if containsVowel(stem) {
			w = replace(w, 3, "")
			fired = true
		}
	}

	if !fired {
		return w
	}

	// Step 1b continuation (first match wins).
	switch {
	case hasSuffix(w, "at"):
		return replace(w, 2, "ate")
	case hasSuffix(w, "bl"):
		return replace(w, 2, "ble")
	case hasSuffix(w, "iz"):
		return replace(w, 2, "ize")
	case endsDoubleC(w):
		last := w[len(w)-1]
		if last != 'l' && last != 's' && last != 'z' {
			return replace(w, 1, "")
		}
	case measure(w) == 1 && endsCVC(w):
		return append(w, 'e')
	}
	return w
}

func step1c(w []byte) []byte {
	if hasSuffix(w, "y") {
		stem := w[:len(w)-1]
		if containsVowel(stem) {
			return replace(w, 1, "i")
		}
	}
	return w
}

// step2 replaces longer suffixes when measure(stem) > 0.
func step2(w []byte) []byte {
	type rule struct{ suffix, repl string }
	rules := []rule{
		{"ational", "ate"},
		{"tional", "tion"},
		{"enci", "ence"},
		{"anci", "ance"},
		{"izer", "ize"},
		{"abli", "able"},
		{"alli", "al"},
		{"entli", "ent"},
		{"eli", "e"},
		{"ousli", "ous"},
		{"ization", "ize"},
		{"ation", "ate"},
		{"ator", "ate"},
		{"alism", "al"},
		{"iveness", "ive"},
		{"fulness", "ful"},
		{"ousness", "ous"},
		{"aliti", "al"},
		{"iviti", "ive"},
		{"biliti", "ble"},
	}
	for _, r := range rules {
		if hasSuffix(w, r.suffix) {
			stem := w[:len(w)-len(r.suffix)]
			if measure(stem) > 0 {
				return replace(w, len(r.suffix), r.repl)
			}
			return w
		}
	}
	return w
}

// step3 replaces suffixes when measure(stem) > 0.
func step3(w []byte) []byte {
	type rule struct{ suffix, repl string }
	rules := []rule{
		{"icate", "ic"},
		{"ative", ""},
		{"alize", "al"},
		{"iciti", "ic"},
		{"ical", "ic"},
		{"ful", ""},
		{"ness", ""},
	}
	for _, r := range rules {
		if hasSuffix(w, r.suffix) {
			stem := w[:len(w)-len(r.suffix)]
			if measure(stem) > 0 {
				return replace(w, len(r.suffix), r.repl)
			}
			return w
		}
	}
	return w
}

// step4 removes suffixes when measure(stem) > 1.
func step4(w []byte) []byte {
	type rule struct {
		suffix  string
		stemEnd string // if non-empty, stem must end in this (for "ion")
	}
	rules := []rule{
		{"al", ""},
		{"ance", ""},
		{"ence", ""},
		{"er", ""},
		{"ic", ""},
		{"able", ""},
		{"ible", ""},
		{"ant", ""},
		{"ement", ""},
		{"ment", ""},
		{"ent", ""},
		{"ion", "st"}, // stem must end in s or t
		{"ou", ""},
		{"ism", ""},
		{"ate", ""},
		{"iti", ""},
		{"ous", ""},
		{"ive", ""},
		{"ize", ""},
	}
	for _, r := range rules {
		if hasSuffix(w, r.suffix) {
			stem := w[:len(w)-len(r.suffix)]
			if measure(stem) > 1 {
				if r.stemEnd != "" {
					// "ion": stem must end in s or t
					last := stem[len(stem)-1]
					if last != 's' && last != 't' {
						return w
					}
				}
				return stem
			}
			return w
		}
	}
	return w
}

func step5a(w []byte) []byte {
	if hasSuffix(w, "e") {
		stem := w[:len(w)-1]
		m := measure(stem)
		if m > 1 || (m == 1 && !endsCVC(stem)) {
			return stem
		}
	}
	return w
}

func step5b(w []byte) []byte {
	if measure(w) > 1 && endsDoubleC(w) && hasSuffix(w, "ll") {
		return replace(w, 1, "")
	}
	return w
}
