package matcher

import (
	"strings"
	"testing"
)

func TestPrefixMatcherMatchesConfiguredGroceryPrefixes(t *testing.T) {
	groceryMatcher, err := NewPrefixMatcher([]string{"HEMKOP", "ICA", "MAXI ICA", "WILLYS", "COOP", "PRESSBYRÅN"})
	if err != nil {
		t.Fatalf("NewPrefixMatcher returned error: %v", err)
	}

	tests := []struct {
		name        string
		description string
		want        bool
	}{
		{name: "case insensitive", description: "ica supermarket stockholm", want: true},
		{name: "maxi ica", description: "MAXI ICA STORMARKNAD", want: true},
		{name: "trim whitespace", description: "  WILLYS GOTEBORG", want: true},
		{name: "swedish character", description: "Pressbyrån Centralen", want: true},
		{name: "folded Swedish character", description: "Hemköp Stockholm", want: true},
		{name: "non grocery", description: "APOTEKET STOCKHOLM", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := groceryMatcher.IsGrocery(tt.description); got != tt.want {
				t.Fatalf("IsGrocery(%q) = %v, want %v", tt.description, got, tt.want)
			}
		})
	}
}

func TestLoadPrefixesIgnoresCommentsAndBlankLines(t *testing.T) {
	prefixes, err := LoadPrefixes(strings.NewReader(`
# comment
ICA

COOP
`))
	if err != nil {
		t.Fatalf("LoadPrefixes returned error: %v", err)
	}

	want := []string{"ICA", "COOP"}
	if len(prefixes) != len(want) {
		t.Fatalf("len(prefixes) = %d, want %d", len(prefixes), len(want))
	}
	for i := range want {
		if prefixes[i] != want[i] {
			t.Fatalf("prefixes[%d] = %q, want %q", i, prefixes[i], want[i])
		}
	}
}
