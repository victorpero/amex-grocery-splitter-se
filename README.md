# amex-grocery-splitter-se

A local CLI tool for finding Swedish grocery transactions in American Express CSV exports and splitting the total between two people.

## Running From The CLI

First, open a terminal and go to this project directory:

```sh
cd ~/dev/amex-grocery-splitter-se
```

The simplest way to run the app while developing is with `go run`:

```sh
go run ./cmd/amex-grocery-splitter "$HOME/Downloads/activity.csv"
```

To show transactions that did not match a grocery-store prefix:

```sh
go run ./cmd/amex-grocery-splitter --show-unmatched "$HOME/Downloads/activity.csv"
```

To process multiple CSV files at once:

```sh
go run ./cmd/amex-grocery-splitter file1.csv file2.csv
```

## Building A Local Binary

If you want to run `amex-grocery-splitter` directly instead of typing `go run ...`, build it first:

```sh
cd ~/dev/amex-grocery-splitter-se
go build -o bin/amex-grocery-splitter ./cmd/amex-grocery-splitter
```

Then run the built binary:

```sh
./bin/amex-grocery-splitter "$HOME/Downloads/activity.csv"
./bin/amex-grocery-splitter --show-unmatched "$HOME/Downloads/activity.csv"
```

## Installed Usage

If the binary is installed somewhere on your `PATH`, you can run it as:

```sh
amex-grocery-splitter transactions.csv
amex-grocery-splitter file1.csv file2.csv
```

Show unmatched transactions for review:

```sh
amex-grocery-splitter --show-unmatched transactions.csv
```

Use signed amounts to inspect refunds or credits:

```sh
amex-grocery-splitter --amount-mode signed transactions.csv
```

Use a custom grocery-prefix file:

```sh
amex-grocery-splitter --stores config/grocery_stores.txt transactions.csv
```

The store file is one prefix per line. Blank lines and lines starting with `#` are ignored.

## Matching

The first version matches grocery transactions by transaction-description prefix. Matching is case-insensitive, trims whitespace, and handles Swedish letters such as Å, Ä, and Ö.

Default grocery prefixes:

- HEMKOP
- ICA
- MAXI ICA
- WILLYS
- COOP
- PRESSBYRÅN

## Amounts

By default, matched transaction amounts are treated as costs using absolute values. This handles AmEx exports where purchases may appear as either positive or negative values.

To preserve signs exactly as they appear in the CSV:

```sh
amex-grocery-splitter --amount-mode signed transactions.csv
```

## CSV support

The parser supports:

- UTF-8 files, including a UTF-8 BOM
- comma or semicolon delimiters
- common English and Swedish column names
- Swedish decimal formats such as `123,45`
- English decimal formats such as `123.45`

Required fields are transaction date, description/name, and amount.

## Development

```sh
go test ./...
go build -o bin/amex-grocery-splitter ./cmd/amex-grocery-splitter
```
