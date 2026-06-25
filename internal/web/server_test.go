package web

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServerGetRendersUploadForm(t *testing.T) {
	server := newTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	if !strings.Contains(body, "Analyze CSV Files") {
		t.Fatalf("body did not contain upload form heading")
	}
	if !strings.Contains(body, "MAXI ICA") {
		t.Fatalf("body did not contain default grocery prefixes")
	}
}

func TestServerPostAnalyzesUploadedCSV(t *testing.T) {
	server := newTestServer(t)
	body, contentType := multipartRequestBody(t, map[string]string{
		"amount_mode":    "absolute",
		"currency":       "SEK",
		"show_unmatched": "on",
		"prefixes":       "ICA\nCOOP",
		"activity.csv":   "Datum;Beskrivning;Belopp\n2026-05-11;COOP RADHUSET;-111,00\n2026-05-12;APOTEKET;50,00\n",
	})
	request := httptest.NewRequest(http.MethodPost, "/", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d\nbody:\n%s", response.Code, http.StatusOK, response.Body.String())
	}
	bodyText := response.Body.String()
	for _, want := range []string{
		"COOP RADHUSET",
		"APOTEKET",
		"SEK 111,00",
		"SEK 55,50",
		"1 matched",
		"1 unmatched",
	} {
		if !strings.Contains(bodyText, want) {
			t.Fatalf("body did not contain %q\nbody:\n%s", want, bodyText)
		}
	}
}

func TestServerPostRequiresCSVFile(t *testing.T) {
	server := newTestServer(t)
	body, contentType := multipartRequestBody(t, map[string]string{
		"amount_mode": "absolute",
		"currency":    "SEK",
		"prefixes":    "ICA\nCOOP",
	})
	request := httptest.NewRequest(http.MethodPost, "/", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), "Choose at least one American Express CSV file") {
		t.Fatalf("body did not contain missing file error")
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	server, err := NewServer(Config{Currency: "SEK"})
	if err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}
	return server
}

func multipartRequestBody(t *testing.T, values map[string]string) (io.Reader, string) {
	t.Helper()
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	for name, value := range values {
		if strings.HasSuffix(name, ".csv") {
			part, err := writer.CreateFormFile("files", name)
			if err != nil {
				t.Fatalf("CreateFormFile returned error: %v", err)
			}
			if _, err := part.Write([]byte(value)); err != nil {
				t.Fatalf("write file part returned error: %v", err)
			}
			continue
		}
		if err := writer.WriteField(name, value); err != nil {
			t.Fatalf("WriteField returned error: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("multipart writer close returned error: %v", err)
	}

	return &buffer, writer.FormDataContentType()
}
