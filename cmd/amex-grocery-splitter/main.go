package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/victorpero/amex-grocery-splitter-se/internal/matcher"
	"github.com/victorpero/amex-grocery-splitter-se/internal/parser"
	"github.com/victorpero/amex-grocery-splitter-se/internal/report"
	"github.com/victorpero/amex-grocery-splitter-se/internal/split"
	"github.com/victorpero/amex-grocery-splitter-se/internal/transaction"
)

type cliConfig struct {
	storesPath    string
	showUnmatched bool
	amountMode    split.AmountMode
	currency      string
	files         []string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	config, err := parseFlags(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	prefixes := matcher.DefaultStorePrefixes()
	if config.storesPath != "" {
		prefixes, err = matcher.LoadPrefixesFile(config.storesPath)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
	}

	groceryMatcher, err := matcher.NewPrefixMatcher(prefixes)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	transactions, err := readTransactions(config.files)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	analysis := report.Analyze(transactions, groceryMatcher, config.amountMode)

	if err := printReport(stdout, config, analysis); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}

func parseFlags(args []string, output io.Writer) (cliConfig, error) {
	flags := flag.NewFlagSet("amex-grocery-splitter", flag.ContinueOnError)
	flags.SetOutput(output)

	storesPath := flags.String("stores", "", "path to grocery store prefix file")
	showUnmatched := flags.Bool("show-unmatched", false, "show non-grocery transactions for review")
	amountModeValue := flags.String("amount-mode", string(split.AmountModeAbsolute), "amount handling: absolute or signed")
	currency := flags.String("currency", "SEK", "currency label used in terminal output")

	if err := flags.Parse(args); err != nil {
		return cliConfig{}, err
	}

	mode, err := split.ParseAmountMode(*amountModeValue)
	if err != nil {
		return cliConfig{}, err
	}

	files := flags.Args()
	if len(files) == 0 {
		return cliConfig{}, fmt.Errorf("at least one CSV file is required\n\nUsage: amex-grocery-splitter [flags] <file.csv> [file2.csv]")
	}

	currencyLabel := strings.TrimSpace(*currency)
	if currencyLabel == "" {
		currencyLabel = "SEK"
	}

	return cliConfig{
		storesPath:    *storesPath,
		showUnmatched: *showUnmatched,
		amountMode:    mode,
		currency:      currencyLabel,
		files:         files,
	}, nil
}

func readTransactions(files []string) ([]transaction.Transaction, error) {
	var transactions []transaction.Transaction
	for _, file := range files {
		parsed, err := parser.ParseFile(file)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		transactions = append(transactions, parsed...)
	}
	return transactions, nil
}

func printReport(output io.Writer, config cliConfig, analysis report.Analysis) error {
	fmt.Fprintln(output, "Matched grocery transactions")
	if len(analysis.Matched) == 0 {
		fmt.Fprintln(output, "No grocery transactions matched.")
	} else {
		printTransactions(output, config.currency, config.amountMode, analysis.Matched)
	}

	fmt.Fprintln(output)
	fmt.Fprintf(output, "Matched transactions: %d\n", len(analysis.Matched))
	fmt.Fprintf(output, "Total grocery amount: %s\n", split.FormatCents(config.currency, analysis.Result.TotalCents))
	fmt.Fprintf(output, "Amount per person:   %s\n", split.FormatHalfCents(config.currency, analysis.Result.TotalCents))

	if config.showUnmatched {
		fmt.Fprintln(output)
		fmt.Fprintln(output, "Unmatched transactions")
		if len(analysis.Unmatched) == 0 {
			fmt.Fprintln(output, "No unmatched transactions.")
		} else {
			printTransactions(output, config.currency, config.amountMode, analysis.Unmatched)
		}
	}

	return nil
}

func printTransactions(output io.Writer, currency string, mode split.AmountMode, transactions []transaction.Transaction) {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "Date\tAmount\tDescription\tSource")
	for _, tx := range transactions {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s:%d\n",
			tx.Date.Format("2006-01-02"),
			split.FormatCents(currency, report.DisplayAmountCents(tx.AmountCents, mode)),
			tx.Description,
			tx.SourceFile,
			tx.SourceLine,
		)
	}
	writer.Flush()
}
