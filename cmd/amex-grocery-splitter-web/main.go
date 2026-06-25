package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/victorpero/amex-grocery-splitter-se/internal/web"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	flag.Parse()

	server, err := web.NewServer(web.Config{
		Currency: "SEK",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	log.Printf("amex-grocery-splitter web UI listening on http://%s", displayAddr(*addr))
	if err := http.ListenAndServe(*addr, server); err != nil {
		log.Fatal(err)
	}
}

func displayAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	if strings.HasPrefix(addr, "0.0.0.0:") {
		return "localhost:" + strings.TrimPrefix(addr, "0.0.0.0:")
	}
	return addr
}
