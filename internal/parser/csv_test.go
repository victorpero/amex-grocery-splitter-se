package parser

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseSemicolonCSVWithSwedishDecimalAmount(t *testing.T) {
	input := strings.NewReader("Datum;Beskrivning;Belopp\n2026-01-02;ICA SUPERMARKET;123,45\n")

	transactions, err := Parse(input, "test.csv")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("len(transactions) = %d, want 1", len(transactions))
	}

	tx := transactions[0]
	if tx.Description != "ICA SUPERMARKET" {
		t.Fatalf("Description = %q, want ICA SUPERMARKET", tx.Description)
	}
	if tx.AmountCents != 12345 {
		t.Fatalf("AmountCents = %d, want 12345", tx.AmountCents)
	}
	wantDate := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if !tx.Date.Equal(wantDate) {
		t.Fatalf("Date = %s, want %s", tx.Date, wantDate)
	}
}

func TestParseCommaCSVWithNegativeEnglishDecimalAmount(t *testing.T) {
	input := strings.NewReader("Date,Description,Amount\n2026-01-02,COOP STOCKHOLM,-45.67\n")

	transactions, err := Parse(input, "test.csv")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(transactions) != 1 {
		t.Fatalf("len(transactions) = %d, want 1", len(transactions))
	}
	if transactions[0].AmountCents != -4567 {
		t.Fatalf("AmountCents = %d, want -4567", transactions[0].AmountCents)
	}
}

func TestParseReportsMissingColumns(t *testing.T) {
	input := strings.NewReader("Date,Description\n2026-01-02,ICA\n")

	_, err := Parse(input, "test.csv")
	if err == nil {
		t.Fatal("Parse returned nil error, want missing columns error")
	}

	var missing MissingColumnsError
	if !errors.As(err, &missing) {
		t.Fatalf("error = %T, want MissingColumnsError", err)
	}
	if len(missing.Missing) != 1 || missing.Missing[0] != "amount" {
		t.Fatalf("Missing = %#v, want [amount]", missing.Missing)
	}
}

func TestParseAmountCentsHandlesCommonFormats(t *testing.T) {
	tests := []struct {
		value string
		want  int64
	}{
		{value: "123,45", want: 12345},
		{value: "123.45", want: 12345},
		{value: "1 234,56 SEK", want: 123456},
		{value: "1,234.56", want: 123456},
		{value: "1.234,56", want: 123456},
		{value: "(123,45)", want: -12345},
		{value: "123,45-", want: -12345},
		{value: "SEK -1 234,56", want: -123456},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := ParseAmountCents(tt.value)
			if err != nil {
				t.Fatalf("ParseAmountCents returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseAmountCents(%q) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}
