package split

import (
	"fmt"
	"strings"

	"github.com/victorpero/amex-grocery-splitter-se/internal/transaction"
)

type AmountMode string

const (
	AmountModeAbsolute AmountMode = "absolute"
	AmountModeSigned   AmountMode = "signed"
)

type Result struct {
	TotalCents     int64
	PerPersonCents int64
	RemainderCents int64
}

func ParseAmountMode(value string) (AmountMode, error) {
	mode := AmountMode(strings.ToLower(strings.TrimSpace(value)))
	switch mode {
	case AmountModeAbsolute, AmountModeSigned:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid amount mode %q, expected %q or %q", value, AmountModeAbsolute, AmountModeSigned)
	}
}

func Calculate(transactions []transaction.Transaction, mode AmountMode) Result {
	total := TotalCents(transactions, mode)
	return Result{
		TotalCents:     total,
		PerPersonCents: total / 2,
		RemainderCents: total % 2,
	}
}

func TotalCents(transactions []transaction.Transaction, mode AmountMode) int64 {
	var total int64
	for _, tx := range transactions {
		amount := tx.AmountCents
		if mode == AmountModeAbsolute {
			amount = abs(amount)
		}
		total += amount
	}
	return total
}

func abs(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func FormatCents(currency string, cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}

	whole := cents / 100
	fraction := cents % 100
	return fmt.Sprintf("%s%s %s,%02d", sign, strings.TrimSpace(currency), formatWhole(whole), fraction)
}

func FormatHalfCents(currency string, totalCents int64) string {
	sign := ""
	if totalCents < 0 {
		sign = "-"
		totalCents = -totalCents
	}

	thousandths := totalCents * 5
	whole := thousandths / 1000
	fraction := thousandths % 1000
	if fraction%10 == 0 {
		return fmt.Sprintf("%s%s %s,%02d", sign, strings.TrimSpace(currency), formatWhole(whole), fraction/10)
	}
	return fmt.Sprintf("%s%s %s,%03d", sign, strings.TrimSpace(currency), formatWhole(whole), fraction)
}

func formatWhole(value int64) string {
	text := fmt.Sprintf("%d", value)
	if len(text) <= 3 {
		return text
	}

	var builder strings.Builder
	firstGroup := len(text) % 3
	if firstGroup == 0 {
		firstGroup = 3
	}
	builder.WriteString(text[:firstGroup])
	for i := firstGroup; i < len(text); i += 3 {
		builder.WriteByte(' ')
		builder.WriteString(text[i : i+3])
	}
	return builder.String()
}
