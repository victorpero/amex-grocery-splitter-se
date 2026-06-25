# amex-grocery-splitter-se

A local CLI tool for finding Swedish grocery transactions in American Express CSV exports and splitting the total between two people.

## Usage

```sh
amex-grocery-splitter transactions.csv
amex-grocery-splitter file1.csv file2.csv
```

From the repository during development:

```sh
go run ./cmd/amex-grocery-splitter transactions.csv
```

Show unmatched transactions for review:

```sh
amex-grocery-splitter --show-unmatched transactions.csv
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
