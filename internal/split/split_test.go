package split

import (
	"testing"

	"github.com/victorpero/amex-grocery-splitter-se/internal/transaction"
)

func TestCalculateAbsoluteModeTreatsTransactionsAsCosts(t *testing.T) {
	transactions := []transaction.Transaction{
		{AmountCents: -10050},
		{AmountCents: 2500},
	}

	result := Calculate(transactions, AmountModeAbsolute)
	if result.TotalCents != 12550 {
		t.Fatalf("TotalCents = %d, want 12550", result.TotalCents)
	}
	if result.PerPersonCents != 6275 {
		t.Fatalf("PerPersonCents = %d, want 6275", result.PerPersonCents)
	}
	if result.RemainderCents != 0 {
		t.Fatalf("RemainderCents = %d, want 0", result.RemainderCents)
	}
}

func TestCalculateSignedModePreservesCSVSigns(t *testing.T) {
	transactions := []transaction.Transaction{
		{AmountCents: -10050},
		{AmountCents: 2500},
	}

	result := Calculate(transactions, AmountModeSigned)
	if result.TotalCents != -7550 {
		t.Fatalf("TotalCents = %d, want -7550", result.TotalCents)
	}
	if result.PerPersonCents != -3775 {
		t.Fatalf("PerPersonCents = %d, want -3775", result.PerPersonCents)
	}
}

func TestFormatHalfCentsShowsHalfCentWhenNeeded(t *testing.T) {
	got := FormatHalfCents("SEK", 10001)
	want := "SEK 50,005"
	if got != want {
		t.Fatalf("FormatHalfCents = %q, want %q", got, want)
	}
}

func TestFormatCentsUsesSwedishStyleSeparators(t *testing.T) {
	got := FormatCents("SEK", 1234567)
	want := "SEK 12 345,67"
	if got != want {
		t.Fatalf("FormatCents = %q, want %q", got, want)
	}
}
