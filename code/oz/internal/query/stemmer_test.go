package query

import "testing"

// TestStem verifies 20 known word/stem pairs against the Porter algorithm.
func TestStem(t *testing.T) {
	cases := []struct {
		word, want string
	}{
		// Step 1a — plural forms
		{"caresses", "caress"},
		{"ponies", "poni"},
		{"ties", "ti"},
		{"caress", "caress"},
		{"cats", "cat"},
		// Step 1b — past/progressive
		{"feed", "feed"},         // eed matched but m("f")=0 → no change
		{"agreed", "agre"},       // eed→ee gives "agree"; step5a removes final e → "agre"
		{"plastered", "plaster"}, // ed rule
		{"bled", "bled"},         // ed, stem "bl" has no vowel → no change
		{"motoring", "motor"},    // ing rule; step4 removes nothing
		{"sing", "sing"},         // ing, stem "s" has no vowel → no change
		{"troubled", "troubl"},   // ed+bl→ble gives "trouble"; step5a removes e → "troubl"
		// Step 1c — y → i
		{"happy", "happi"},
		{"sky", "sky"}, // stem "sk" has no vowel → no change
		// Compound steps
		{"generalize", "gener"}, // step3 alize→al gives "general"; step4 al+m>1 → "gener"
		{"electric", "electr"},  // step4 ic, m("electr")=2>1
		{"connection", "connect"}, // step4 ion, stem "connect" ends in t, m=2>1
		{"running", "run"},   // step1b double nn → "run", stop
		{"hopping", "hop"},   // step1b double pp → "hop", stop
		{"matting", "mat"},   // step1b double tt → "mat", stop
	}

	for _, tc := range cases {
		got := Stem(tc.word)
		if got != tc.want {
			t.Errorf("Stem(%q) = %q, want %q", tc.word, got, tc.want)
		}
	}
}

// TestStem_Deterministic verifies that stemming the same word twice gives the same result.
func TestStem_Deterministic(t *testing.T) {
	words := []string{"implementation", "testing", "architecture", "dependencies", "configuration"}
	for _, w := range words {
		a := Stem(w)
		b := Stem(w)
		if a != b {
			t.Errorf("Stem(%q) not deterministic: %q vs %q", w, a, b)
		}
	}
}
