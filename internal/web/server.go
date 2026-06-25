package web

import (
	"fmt"
	"html/template"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/victorpero/amex-grocery-splitter-se/internal/matcher"
	"github.com/victorpero/amex-grocery-splitter-se/internal/parser"
	"github.com/victorpero/amex-grocery-splitter-se/internal/report"
	"github.com/victorpero/amex-grocery-splitter-se/internal/split"
	"github.com/victorpero/amex-grocery-splitter-se/internal/transaction"
)

const maxUploadBytes = 32 << 20

type Config struct {
	Currency string
}

type Server struct {
	config   Config
	template *template.Template
}

type pageData struct {
	Form          formData
	Error         string
	HasResult     bool
	TotalFiles    int
	TotalRows     int
	Matched       []viewTransaction
	Unmatched     []viewTransaction
	ShowUnmatched bool
	TotalAmount   string
	PerPerson     string
}

type formData struct {
	AmountMode string
	Currency   string
	Prefixes   string
}

type viewTransaction struct {
	Date        string
	Description string
	Amount      string
	Source      string
}

func NewServer(config Config) (*Server, error) {
	if strings.TrimSpace(config.Currency) == "" {
		config.Currency = "SEK"
	}

	parsed, err := template.New("page").Parse(pageTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse web template: %w", err)
	}

	return &Server{
		config:   config,
		template: parsed,
	}, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path != "/":
		http.NotFound(w, r)
	case r.Method == http.MethodGet:
		s.render(w, http.StatusOK, s.emptyPage())
	case r.Method == http.MethodPost:
		s.handleAnalyze(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) emptyPage() pageData {
	return pageData{
		Form: formData{
			AmountMode: string(split.AmountModeAbsolute),
			Currency:   s.config.Currency,
			Prefixes:   strings.Join(matcher.DefaultStorePrefixes(), "\n"),
		},
	}
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		data := s.emptyPage()
		data.Error = fmt.Sprintf("Could not read uploaded files: %v", err)
		s.render(w, http.StatusBadRequest, data)
		return
	}

	form := formData{
		AmountMode: strings.TrimSpace(r.FormValue("amount_mode")),
		Currency:   strings.TrimSpace(r.FormValue("currency")),
		Prefixes:   strings.TrimSpace(r.FormValue("prefixes")),
	}
	if form.AmountMode == "" {
		form.AmountMode = string(split.AmountModeAbsolute)
	}
	if form.Currency == "" {
		form.Currency = s.config.Currency
	}
	if form.Prefixes == "" {
		form.Prefixes = strings.Join(matcher.DefaultStorePrefixes(), "\n")
	}

	data := pageData{
		Form:          form,
		ShowUnmatched: r.FormValue("show_unmatched") == "on",
	}

	amountMode, err := split.ParseAmountMode(form.AmountMode)
	if err != nil {
		data.Error = err.Error()
		s.render(w, http.StatusBadRequest, data)
		return
	}

	prefixes, err := matcher.LoadPrefixes(strings.NewReader(form.Prefixes))
	if err != nil {
		data.Error = fmt.Sprintf("Could not read grocery prefixes: %v", err)
		s.render(w, http.StatusBadRequest, data)
		return
	}
	groceryMatcher, err := matcher.NewPrefixMatcher(prefixes)
	if err != nil {
		data.Error = err.Error()
		s.render(w, http.StatusBadRequest, data)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		data.Error = "Choose at least one American Express CSV file."
		s.render(w, http.StatusBadRequest, data)
		return
	}

	transactions, err := parseUploadedFiles(files)
	if err != nil {
		data.Error = err.Error()
		s.render(w, http.StatusBadRequest, data)
		return
	}

	analysis := report.Analyze(transactions, groceryMatcher, amountMode)
	data.HasResult = true
	data.TotalFiles = len(files)
	data.TotalRows = len(transactions)
	data.Matched = toViewTransactions(analysis.Matched, form.Currency, amountMode)
	data.Unmatched = toViewTransactions(analysis.Unmatched, form.Currency, amountMode)
	data.TotalAmount = split.FormatCents(form.Currency, analysis.Result.TotalCents)
	data.PerPerson = split.FormatHalfCents(form.Currency, analysis.Result.TotalCents)

	s.render(w, http.StatusOK, data)
}

func parseUploadedFiles(files []*multipart.FileHeader) ([]transaction.Transaction, error) {
	transactions := make([]transaction.Transaction, 0)
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			return nil, fmt.Errorf("%s: open uploaded CSV: %w", header.Filename, err)
		}

		parsed, parseErr := parser.Parse(file, header.Filename)
		closeErr := file.Close()
		if parseErr != nil {
			return nil, fmt.Errorf("%s: %w", header.Filename, parseErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("%s: close uploaded CSV: %w", header.Filename, closeErr)
		}
		transactions = append(transactions, parsed...)
	}
	return transactions, nil
}

func toViewTransactions(transactions []transaction.Transaction, currency string, amountMode split.AmountMode) []viewTransaction {
	view := make([]viewTransaction, 0, len(transactions))
	for _, tx := range transactions {
		view = append(view, viewTransaction{
			Date:        tx.Date.Format("2006-01-02"),
			Description: tx.Description,
			Amount:      split.FormatCents(currency, report.DisplayAmountCents(tx.AmountCents, amountMode)),
			Source:      fmt.Sprintf("%s:%d", tx.SourceFile, tx.SourceLine),
		})
	}
	return view
}

func (s *Server) render(w http.ResponseWriter, status int, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := s.template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

const pageTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>AmEx Grocery Splitter</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f7f7f4;
      --panel: #ffffff;
      --ink: #202124;
      --muted: #666d75;
      --line: #d9ddd9;
      --accent: #0f766e;
      --accent-strong: #115e59;
      --danger-bg: #fff1f0;
      --danger-line: #f1b5ad;
      --danger-text: #9f1d16;
      --header: #ecefeb;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--ink);
      font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      line-height: 1.45;
    }
    header {
      background: var(--panel);
      border-bottom: 1px solid var(--line);
    }
    .wrap {
      width: min(1180px, calc(100% - 32px));
      margin: 0 auto;
    }
    .topbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 20px;
      padding: 22px 0;
    }
    h1 {
      margin: 0;
      font-size: 24px;
      font-weight: 700;
    }
    .subtle {
      color: var(--muted);
      font-size: 14px;
    }
    main {
      padding: 24px 0 48px;
    }
    .layout {
      display: grid;
      grid-template-columns: 360px minmax(0, 1fr);
      gap: 24px;
      align-items: start;
    }
    .panel {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 8px;
    }
    .panel h2 {
      margin: 0;
      padding: 16px 18px;
      border-bottom: 1px solid var(--line);
      font-size: 16px;
    }
    .form {
      padding: 18px;
      display: grid;
      gap: 16px;
    }
    label {
      display: grid;
      gap: 7px;
      color: var(--muted);
      font-size: 13px;
      font-weight: 600;
    }
    input[type="file"],
    input[type="text"],
    select,
    textarea {
      width: 100%;
      border: 1px solid var(--line);
      border-radius: 6px;
      color: var(--ink);
      background: #fff;
      font: inherit;
      font-size: 14px;
      padding: 10px 11px;
    }
    textarea {
      min-height: 150px;
      resize: vertical;
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    }
    .row {
      display: grid;
      grid-template-columns: 1fr 96px;
      gap: 12px;
    }
    .check {
      display: flex;
      align-items: center;
      gap: 9px;
      color: var(--ink);
      font-size: 14px;
      font-weight: 500;
    }
    .check input {
      inline-size: 16px;
      block-size: 16px;
    }
    button {
      border: 0;
      border-radius: 6px;
      background: var(--accent);
      color: #fff;
      cursor: pointer;
      font: inherit;
      font-weight: 700;
      padding: 11px 14px;
    }
    button:hover {
      background: var(--accent-strong);
    }
    .error {
      margin-bottom: 18px;
      border: 1px solid var(--danger-line);
      border-radius: 8px;
      background: var(--danger-bg);
      color: var(--danger-text);
      padding: 13px 15px;
      font-weight: 600;
    }
    .summary {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      border-bottom: 1px solid var(--line);
    }
    .metric {
      padding: 16px 18px;
      border-right: 1px solid var(--line);
    }
    .metric:last-child {
      border-right: 0;
    }
    .metric span {
      display: block;
      color: var(--muted);
      font-size: 12px;
      font-weight: 700;
      text-transform: uppercase;
    }
    .metric strong {
      display: block;
      margin-top: 5px;
      font-size: 20px;
    }
    .table-block {
      padding: 18px;
    }
    .section-head {
      display: flex;
      align-items: baseline;
      justify-content: space-between;
      gap: 16px;
      margin-bottom: 10px;
    }
    .section-head h3 {
      margin: 0;
      font-size: 15px;
    }
    .table-wrap {
      overflow-x: auto;
      border: 1px solid var(--line);
      border-radius: 8px;
    }
    table {
      width: 100%;
      min-width: 720px;
      border-collapse: collapse;
      background: #fff;
    }
    th,
    td {
      padding: 10px 12px;
      border-bottom: 1px solid var(--line);
      text-align: left;
      font-size: 14px;
      vertical-align: top;
    }
    th {
      background: var(--header);
      color: #3c434a;
      font-size: 12px;
      text-transform: uppercase;
    }
    tr:last-child td {
      border-bottom: 0;
    }
    .amount {
      white-space: nowrap;
      text-align: right;
      font-variant-numeric: tabular-nums;
    }
    .empty {
      padding: 36px 18px;
      color: var(--muted);
      text-align: center;
    }
    @media (max-width: 860px) {
      .topbar {
        align-items: flex-start;
        flex-direction: column;
      }
      .layout {
        grid-template-columns: 1fr;
      }
      .summary {
        grid-template-columns: repeat(2, minmax(0, 1fr));
      }
      .metric:nth-child(2) {
        border-right: 0;
      }
      .metric:nth-child(-n+2) {
        border-bottom: 1px solid var(--line);
      }
    }
    @media (max-width: 520px) {
      .wrap {
        width: min(100% - 20px, 1180px);
      }
      .row,
      .summary {
        grid-template-columns: 1fr;
      }
      .metric {
        border-right: 0;
        border-bottom: 1px solid var(--line);
      }
      .metric:last-child {
        border-bottom: 0;
      }
    }
  </style>
</head>
<body>
  <header>
    <div class="wrap topbar">
      <div>
        <h1>AmEx Grocery Splitter</h1>
        <div class="subtle">Upload American Express CSV exports and split matched Swedish grocery purchases.</div>
      </div>
      <div class="subtle">Files are processed by this server and are not stored.</div>
    </div>
  </header>
  <main>
    <div class="wrap">
      {{if .Error}}<div class="error">{{.Error}}</div>{{end}}
      <div class="layout">
        <section class="panel">
          <h2>Analyze CSV Files</h2>
          <form class="form" method="post" enctype="multipart/form-data">
            <label>
              CSV files
              <input type="file" name="files" accept=".csv,text/csv" multiple required>
            </label>
            <div class="row">
              <label>
                Amount mode
                <select name="amount_mode">
                  <option value="absolute" {{if eq .Form.AmountMode "absolute"}}selected{{end}}>Absolute costs</option>
                  <option value="signed" {{if eq .Form.AmountMode "signed"}}selected{{end}}>Signed CSV amounts</option>
                </select>
              </label>
              <label>
                Currency
                <input type="text" name="currency" value="{{.Form.Currency}}">
              </label>
            </div>
            <label class="check">
              <input type="checkbox" name="show_unmatched" {{if .ShowUnmatched}}checked{{end}}>
              Show unmatched transactions
            </label>
            <label>
              Grocery prefixes
              <textarea name="prefixes" spellcheck="false">{{.Form.Prefixes}}</textarea>
            </label>
            <button type="submit">Analyze</button>
          </form>
        </section>

        <section class="panel">
          {{if .HasResult}}
            <div class="summary">
              <div class="metric"><span>Files</span><strong>{{.TotalFiles}}</strong></div>
              <div class="metric"><span>Rows</span><strong>{{.TotalRows}}</strong></div>
              <div class="metric"><span>Total</span><strong>{{.TotalAmount}}</strong></div>
              <div class="metric"><span>Each Pays</span><strong>{{.PerPerson}}</strong></div>
            </div>
            <div class="table-block">
              <div class="section-head">
                <h3>Matched Grocery Transactions</h3>
                <div class="subtle">{{len .Matched}} matched</div>
              </div>
              {{template "table" .Matched}}
            </div>
            {{if .ShowUnmatched}}
              <div class="table-block">
                <div class="section-head">
                  <h3>Unmatched Transactions</h3>
                  <div class="subtle">{{len .Unmatched}} unmatched</div>
                </div>
                {{template "table" .Unmatched}}
              </div>
            {{end}}
          {{else}}
            <div class="empty">Choose one or more CSV files to see matched grocery transactions and the split amount.</div>
          {{end}}
        </section>
      </div>
    </div>
  </main>
</body>
</html>

{{define "table"}}
  {{if .}}
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Date</th>
            <th>Description</th>
            <th class="amount">Amount</th>
            <th>Source</th>
          </tr>
        </thead>
        <tbody>
          {{range .}}
            <tr>
              <td>{{.Date}}</td>
              <td>{{.Description}}</td>
              <td class="amount">{{.Amount}}</td>
              <td>{{.Source}}</td>
            </tr>
          {{end}}
        </tbody>
      </table>
    </div>
  {{else}}
    <div class="empty">No transactions in this group.</div>
  {{end}}
{{end}}
`
