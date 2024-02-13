package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/admtnnr/fetch"
)

var (
	port = flag.Int("port", 8080, "port of API server")
)

func main() {
	fmt.Fprintf(os.Stderr, "starting Fetch API server\n")

	ctx := context.Background()
	api := fetch.NewAPI()

	srv := http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      api,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go srv.ListenAndServe()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGTERM)

	<-sigc

	fmt.Fprintf(os.Stderr, "shutting down Fetch API server\n")

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to shutdown Fetch API server: %v\n", err)
		os.Exit(1)
	}
}
