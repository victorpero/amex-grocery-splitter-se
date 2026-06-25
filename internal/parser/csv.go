package parser

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/victorpero/amex-grocery-splitter-se/internal/transaction"
)

type MissingColumnsError struct {
	Missing []string
	Found   []string
}

func (e MissingColumnsError) Error() string {
	return fmt.Sprintf("missing required column(s): %s. Found columns: %s", strings.Join(e.Missing, ", "), strings.Join(e.Found, ", "))
}

func ParseFile(path string) ([]transaction.Transaction, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open CSV file: %w", err)
	}
	defer file.Close()

	return Parse(file, path)
}

func Parse(reader io.Reader, sourceName string) ([]transaction.Transaction, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read CSV data: %w", err)
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	delimiter := DetectDelimiter(data)
	csvReader := csv.NewReader(bytes.NewReader(data))
	csvReader.Comma = delimiter
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = true
	csvReader.LazyQuotes = true

	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}
	columns, err := findColumns(header)
	if err != nil {
		return nil, err
	}

	transactions := make([]transaction.Transaction, 0)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			var parseErr *csv.ParseError
			if errors.As(err, &parseErr) {
				return nil, fmt.Errorf("read CSV record near line %d: %w", parseErr.Line, err)
			}
			return nil, fmt.Errorf("read CSV record: %w", err)
		}
		if isEmptyRecord(record) {
			continue
		}

		line, _ := csvReader.FieldPos(0)
		tx, err := parseRecord(record, columns, sourceName, line)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

func DetectDelimiter(data []byte) rune {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		semicolons := countUnquoted(line, ';')
		commas := countUnquoted(line, ',')
		if semicolons > commas {
			return ';'
		}
		return ','
	}
	return ','
}

type columns struct {
	date        int
	description int
	amount      int
}

func findColumns(header []string) (columns, error) {
	normalized := make([]string, len(header))
	for i, column := range header {
		normalized[i] = normalizeHeader(column)
	}

	found := columns{
		date:        findColumn(normalized, dateColumnAliases),
		description: findColumn(normalized, descriptionColumnAliases),
		amount:      findColumn(normalized, amountColumnAliases),
	}

	var missing []string
	if found.date == -1 {
		missing = append(missing, "date")
	}
	if found.description == -1 {
		missing = append(missing, "description/name")
	}
	if found.amount == -1 {
		missing = append(missing, "amount")
	}
	if len(missing) > 0 {
		return columns{}, MissingColumnsError{
			Missing: missing,
			Found:   trimHeader(header),
		}
	}
	return found, nil
}

var dateColumnAliases = map[string]struct{}{
	"date":            {},
	"datum":           {},
	"transactiondate": {},
	"posteddate":      {},
	"postdate":        {},
}

var descriptionColumnAliases = map[string]struct{}{
	"description":             {},
	"name":                    {},
	"namn":                    {},
	"beskrivning":             {},
	"transactiondescription":  {},
	"transactiondescriptions": {},
	"merchant":                {},
	"merchantname":            {},
}

var amountColumnAliases = map[string]struct{}{
	"amount":            {},
	"belopp":            {},
	"summa":             {},
	"transactionamount": {},
}

func findColumn(header []string, aliases map[string]struct{}) int {
	for index, column := range header {
		if _, ok := aliases[column]; ok {
			return index
		}
	}
	return -1
}

func parseRecord(record []string, cols columns, sourceName string, line int) (transaction.Transaction, error) {
	description, err := field(record, cols.description, "description/name", sourceName, line)
	if err != nil {
		return transaction.Transaction{}, err
	}
	dateValue, err := field(record, cols.date, "date", sourceName, line)
	if err != nil {
		return transaction.Transaction{}, err
	}
	amountValue, err := field(record, cols.amount, "amount", sourceName, line)
	if err != nil {
		return transaction.Transaction{}, err
	}

	date, err := ParseDate(dateValue)
	if err != nil {
		return transaction.Transaction{}, fmt.Errorf("%s line %d: invalid date %q: %w", sourceName, line, dateValue, err)
	}

	amountCents, err := ParseAmountCents(amountValue)
	if err != nil {
		return transaction.Transaction{}, fmt.Errorf("%s line %d: invalid amount %q: %w", sourceName, line, amountValue, err)
	}

	return transaction.Transaction{
		Date:        date,
		Description: strings.TrimSpace(description),
		AmountCents: amountCents,
		SourceFile:  sourceName,
		SourceLine:  line,
	}, nil
}

func field(record []string, index int, name string, sourceName string, line int) (string, error) {
	if index >= len(record) {
		return "", fmt.Errorf("%s line %d: missing %s field", sourceName, line, name)
	}
	return record[index], nil
}

func ParseDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	layouts := []string{
		"2006-01-02",
		"2006/01/02",
		"01/02/2006",
		"1/2/2006",
		"02/01/2006",
		"2/1/2006",
		"02.01.2006",
		"2.1.2006",
		"02 Jan 2006",
		"2 Jan 2006",
		"Jan 02, 2006",
		"January 02, 2006",
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("expected a recognizable date format such as 2006-01-02 or 01/02/2006")
}

func ParseAmountCents(value string) (int64, error) {
	original := value
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("amount is empty")
	}

	negative := false
	if strings.Contains(value, "(") && strings.Contains(value, ")") {
		negative = true
	}
	if strings.ContainsAny(value, "-−") {
		negative = true
	}

	var cleaned strings.Builder
	for _, r := range value {
		switch {
		case unicode.IsDigit(r), r == ',' || r == '.':
			cleaned.WriteRune(r)
		case r == '-' || r == '−' || r == '+' || unicode.IsSpace(r) || r == '\'' || r == '(' || r == ')':
			continue
		case unicode.IsLetter(r):
			continue
		default:
			continue
		}
	}

	number := cleaned.String()
	if number == "" {
		return 0, fmt.Errorf("amount %q does not contain digits", original)
	}

	separator := decimalSeparator(number)
	integerPart := number
	fractionalPart := ""
	if separator != 0 {
		index := strings.LastIndex(number, string(separator))
		integerPart = number[:index]
		fractionalPart = number[index+1:]
	}

	integerDigits := digitsOnly(integerPart)
	fractionalDigits := digitsOnly(fractionalPart)
	if integerDigits == "" {
		integerDigits = "0"
	}
	if len(fractionalDigits) > 2 {
		return 0, fmt.Errorf("amount has more than two decimal places")
	}
	for len(fractionalDigits) < 2 {
		fractionalDigits += "0"
	}

	whole, err := strconv.ParseInt(integerDigits, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse whole amount: %w", err)
	}
	cents, err := strconv.ParseInt(fractionalDigits, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse fractional amount: %w", err)
	}

	total := whole*100 + cents
	if negative {
		total = -total
	}
	return total, nil
}

func decimalSeparator(number string) rune {
	lastComma := strings.LastIndex(number, ",")
	lastDot := strings.LastIndex(number, ".")
	switch {
	case lastComma == -1 && lastDot == -1:
		return 0
	case lastComma > lastDot:
		if hasTwoDigitsAfter(number, lastComma) {
			return ','
		}
	case lastDot > lastComma:
		if hasTwoDigitsAfter(number, lastDot) {
			return '.'
		}
	}
	return 0
}

func hasTwoDigitsAfter(value string, index int) bool {
	digits := 0
	for _, r := range value[index+1:] {
		if unicode.IsDigit(r) {
			digits++
		}
	}
	return digits == 2
}

func digitsOnly(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func countUnquoted(line string, target rune) int {
	inQuotes := false
	count := 0
	for _, r := range line {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case target:
			if !inQuotes {
				count++
			}
		}
	}
	return count
}

func normalizeHeader(value string) string {
	value = strings.TrimPrefix(strings.TrimSpace(value), "\ufeff")
	var builder strings.Builder
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func trimHeader(header []string) []string {
	trimmed := make([]string, len(header))
	for i, value := range header {
		trimmed[i] = strings.TrimSpace(value)
	}
	return trimmed
}

func isEmptyRecord(record []string) bool {
	for _, field := range record {
		if strings.TrimSpace(field) != "" {
			return false
		}
	}
	return true
}
