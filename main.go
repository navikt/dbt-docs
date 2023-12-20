package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/navikt/dbt-docs/pkg/api"
	"github.com/navikt/dbt-docs/pkg/bigquery"
)

var (
	bucketName      string
	bigqueryProject string
	bigqueryDataset string
	bigqueryTable   string
	noMetrics       bool
)

func init() {
	flag.StringVar(&bucketName, "bucket-name", os.Getenv("DBT_DOCS_BUCKET"), "The dbt docs bucket")
	flag.StringVar(&bigqueryProject, "bigquery-project", os.Getenv("GCP_TEAM_PROJECT_ID"), "The GCP bigquery project for metrics")
	flag.StringVar(&bigqueryDataset, "bigquery-dataset", os.Getenv("BIGQUERY_DATASET"), "The GCP bigquery dataset for metrics")
	flag.StringVar(&bigqueryTable, "bigquery-table", os.Getenv("BIGQUERY_TABLE"), "The GCP bigquery table for metrics")
	flag.BoolVar(&noMetrics, "no-metrics", os.Getenv("NO_METRICS") == "true", "Disable metrics")
}

func main() {
	flag.Parse()
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if bucketName == "" {
		logger.Error("missing bucket name")
		os.Exit(1)
	}

	bq, err := bigquery.New(ctx, bigqueryProject, bigqueryDataset, bigqueryTable, noMetrics)
	if err != nil {
		logger.Error("creating bigquery client", "error", err)
		os.Exit(1)
	}

	router, err := api.New(ctx, bucketName, bq, logger.With("subsystem", "api"))
	if err != nil {
		logger.Error("creating api", "error", err)
		os.Exit(1)
	}

	logger.Info("starting server")
	server := http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	if err := server.ListenAndServe(); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
