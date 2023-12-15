package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/navikt/dbt-docs/pkg/api"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	router, err := api.New(ctx, "nada-dbt-docs-dev")
	if err != nil {
		logger.Error("creating api", "error", err)
		os.Exit(1)
	}

	server := http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	if err := server.ListenAndServe(); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
