package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/openilink/openilink-hub/internal/provider/ilink/mockserver"
)

func main() {
	listen := flag.String("listen", ":9900", "listen address")
	flag.Parse()

	srv := mockserver.NewHTTPServer()
	fmt.Printf("iLink Mock Server running on http://localhost%s\n", *listen)
	fmt.Println("  POST /mock/inbound        — inject inbound message")
	fmt.Println("  GET  /mock/sent           — view sent messages")
	fmt.Println("  POST /mock/qr/scan        — simulate QR scan")
	fmt.Println("  POST /mock/qr/confirm     — confirm QR binding")
	fmt.Println("  POST /mock/session/expire  — expire session")
	fmt.Println("  POST /mock/reset          — reset all state")
	if err := http.ListenAndServe(*listen, srv.Handler()); err != nil {
		slog.Error("server error", "err", err)
	}
}
