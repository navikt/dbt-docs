package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/navikt/dbt-docs/pkg/api"
)

var bucketName string

func init() {
	flag.StringVar(&bucketName, "bucket-name", os.Getenv("DBT_DOCS_BUCKET"), "The dbt docs bucket")
}

func main() {
	flag.Parse()
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	router, err := api.New(ctx, bucketName, logger.With("subsystem", "api"))
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
