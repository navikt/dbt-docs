package gcs

import (
	"context"
	"errors"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type GCSClient struct {
	client *storage.Client
	bucket string
}

func New(ctx context.Context, bucket string) (*GCSClient, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &GCSClient{
		client: client,
		bucket: bucket,
	}, nil
}

func (g *GCSClient) ListTeamsAndDocsInBucket(ctx context.Context) map[string][]string {
	teamsDocsMap := map[string][]string{}
	objects := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{
		Prefix: "docs/",
	})
	for {
		o, err := objects.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		pathParts := strings.Split(o.Name, "/")
		if len(pathParts) >= 3 && !contains(teamsDocsMap, pathParts[1], pathParts[2]) {
			teamsDocsMap[pathParts[1]] = append(teamsDocsMap[pathParts[1]], pathParts[2])
		}
	}

	return teamsDocsMap
}

func (g *GCSClient) GetFile(ctx context.Context, filePath string) ([]byte, error) {
	objReader, err := g.client.Bucket(g.bucket).Object(filePath).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(objReader)
}

func (g *GCSClient) DeleteFilesWithPrefix(ctx context.Context, prefix string) error {
	files := g.ListFilesWithPrefix(ctx, prefix)

	for _, file := range files {
		if err := g.client.Bucket(g.bucket).Object(file).Delete(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (g *GCSClient) ListFilesWithPrefix(ctx context.Context, prefix string) []string {
	files := []string{}
	dbts := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})
	for {
		o, err := dbts.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		files = append(files, o.Name)
	}

	return files
}

func (g *GCSClient) UploadFile(ctx context.Context, filePath string, content []byte) error {
	writer := g.client.Bucket(g.bucket).Object(filePath).NewWriter(ctx)
	_, err := writer.Write(content)
	if err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	return nil
}

func contains(teamsDocsMap map[string][]string, team, docID string) bool {
	if _, ok := teamsDocsMap[team]; ok {
		for _, existing := range teamsDocsMap[team] {
			if existing == docID {
				return true
			}
		}
	}

	return false
}
