package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/navikt/dbt-docs/pkg/api"
)

var (
	bucketName      string
	bigqueryProject string
	bigqueryDataset string
	bigqueryTable   string
)

func init() {
	flag.StringVar(&bucketName, "bucket-name", os.Getenv("DBT_DOCS_BUCKET"), "The dbt docs bucket")
	flag.StringVar(&bigqueryProject, "bigquery-project", os.Getenv("GCP_TEAM_PROJECT_ID"), "The GCP bigquery project for metrics")
	flag.StringVar(&bigqueryDataset, "bigquery-dataset", os.Getenv("BIGQUERY_DATASET"), "The GCP bigquery dataset for metrics")
	flag.StringVar(&bigqueryTable, "bigquery-table", os.Getenv("BIGQUERY_TABLE"), "The GCP bigquery table for metrics")
}

func main() {
	flag.Parse()
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// bq, err := bigquery.New(ctx, bigqueryProject, bigqueryDataset, bigqueryTable)
	// if err != nil {
	// 	logger.Error("creating bigquery client", "error", err)
	// 	os.Exit(1)
	// }

	router, err := api.New(ctx, bucketName, nil, logger.With("subsystem", "api"))
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
