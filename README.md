# amex-grocery-splitter-se

A local CLI tool for reading American Express CSV exports, finding Swedish grocery transactions, summing them, and splitting the total between two people.

## Matching

The first version matches grocery transactions by transaction-description prefix. Matching is case-insensitive, trims whitespace, and handles Swedish letters such as Å, Ä, and Ö.

Default grocery prefixes:

- HEMKOP
- ICA
- MAXI ICA
- WILLYS
- COOP
- PRESSBYRÅN

## Running The Local Web GUI

Start the local web UI:

```sh
go run ./cmd/amex-grocery-splitter-web
```

Then open this address in your browser:

```text
http://localhost:8080
```

The web UI lets you upload one or more AmEx CSV files, edit grocery prefixes, choose signed or absolute amount handling, and review matched and unmatched transactions.

By default, the web server only listens on your own machine at `127.0.0.1:8080`. To make it reachable from other devices on your local network later, bind it to all network interfaces:

```sh
go run ./cmd/amex-grocery-splitter-web -addr 0.0.0.0:8080
```

For a container or home-server deployment, use the same `-addr 0.0.0.0:8080` setting and publish port `8080`.

## Running From The CLI

First, open a terminal and go to this project directory:

```sh
cd ~/dev/amex-grocery-splitter-se
```

On Windows PowerShell, the same step looks like:

```powershell
cd $HOME\dev\amex-grocery-splitter-se
```

The simplest way to run the app while developing is with `go run`:

```sh
go run ./cmd/amex-grocery-splitter "$HOME/Downloads/activity.csv"
```

On Windows PowerShell:

```powershell
go run ./cmd/amex-grocery-splitter "$HOME\Downloads\activity.csv"
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

If you want to avoid typing `go run ...`, build a local binary first.

macOS and Linux:

```sh
cd ~/dev/amex-grocery-splitter-se
go build -o bin/amex-grocery-splitter ./cmd/amex-grocery-splitter
```

To build the web UI binary:

```sh
go build -o bin/amex-grocery-splitter-web ./cmd/amex-grocery-splitter-web
```

Then run the built binary:

```sh
./bin/amex-grocery-splitter "$HOME/Downloads/activity.csv"
./bin/amex-grocery-splitter --show-unmatched "$HOME/Downloads/activity.csv"
```

Windows PowerShell:

```powershell
cd $HOME\dev\amex-grocery-splitter-se
go build -o bin\amex-grocery-splitter.exe ./cmd/amex-grocery-splitter
```

Then run the built binary:

```powershell
.\bin\amex-grocery-splitter.exe "$HOME\Downloads\activity.csv"
.\bin\amex-grocery-splitter.exe --show-unmatched "$HOME\Downloads\activity.csv"
```

The `./bin/...` or `.\bin\...` prefix is important. It tells your shell to run the binary from this project folder. Without installing the binary on your `PATH`, this will not work:

```sh
amex-grocery-splitter "$HOME/Downloads/activity.csv"
```

## Installing On Your PATH

Install the command if you want to run `amex-grocery-splitter` from any folder without the `./bin/...` prefix:

```sh
go install ./cmd/amex-grocery-splitter
```

Go installs binaries into `GOPATH/bin`. By default, that is usually:

- macOS and Linux: `$HOME/go/bin`
- Windows: `%USERPROFILE%\go\bin`

If that directory is on your `PATH`, this works from any folder:

```sh
amex-grocery-splitter "$HOME/Downloads/activity.csv"
```

On Windows PowerShell:

```powershell
amex-grocery-splitter "$HOME\Downloads\activity.csv"
```

If macOS or Linux says `command not found`, add Go's bin directory to your shell path.

For zsh, which is the default shell on modern macOS:

```sh
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

For bash:

```sh
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

If Windows says the command is not recognized, add this folder to your user `Path` environment variable:

```text
%USERPROFILE%\go\bin
```

Then open a new PowerShell window.

## Usage Examples

Once you are using either `go run`, a local binary, or an installed binary, the available flags are the same. These examples assume the binary is installed on your `PATH`:

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
