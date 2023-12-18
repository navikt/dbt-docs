package bigquery

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"
)

type BigQuery struct {
	client  *bigquery.Client
	project string
	dataset string
	table   string
}

type viewTableEntry struct {
	Timestamp bigquery.NullTimestamp `json:"timestamp"`
	URL       string                 `json:"url"`
}

func New(ctx context.Context, project, dataset, table string) (*BigQuery, error) {
	client, err := bigquery.NewClient(ctx, project)
	if err != nil {
		return nil, err
	}

	if err := createMetricsTable(ctx, client, dataset, table); err != nil {
		return nil, err
	}

	return &BigQuery{
		client:  client,
		project: project,
		dataset: dataset,
		table:   table,
	}, nil
}

func (bq *BigQuery) RegisterView(ctx context.Context, url string) error {
	table := bq.client.DatasetInProject(bq.project, bq.dataset).Table(bq.table)
	inserter := table.Inserter()

	viewEntry := viewTableEntry{
		Timestamp: bigquery.NullTimestamp{Timestamp: time.Now(), Valid: true},
		URL:       url,
	}

	return inserter.Put(ctx, viewEntry)
}

func createMetricsTable(ctx context.Context, client *bigquery.Client, datasetName, tableName string) error {
	dataset := client.Dataset(datasetName)

	err := dataset.Table(tableName).Create(ctx, &bigquery.TableMetadata{
		Name:     "Visninger av dbt dokumentasjon",
		Location: "europe-north1",
		Labels: map[string]string{
			"created-by": "dbt-docs",
			"team":       "nada",
		},
		Description: "Tabell over visninger av dbt dokumentasjon",
		Schema: bigquery.Schema{
			&bigquery.FieldSchema{Name: "timestamp", Type: bigquery.TimestampFieldType, Required: true},
			&bigquery.FieldSchema{Name: "url", Type: bigquery.StringFieldType, Required: true},
		},
	})
	if err != nil {
		var e *googleapi.Error
		if ok := errors.As(err, &e); ok {
			if e.Code == 409 {
				// already exists
				return nil
			}
			return err
		}
	}

	return nil
}
