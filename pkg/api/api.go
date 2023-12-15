package api

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/labstack/echo/v4"
	"google.golang.org/api/iterator"
)

type Templates struct {
	templates map[string]*template.Template
}

func (t *Templates) Render(w io.Writer, name string, data any, c echo.Context) error {
	template, ok := t.templates[name]
	if !ok {
		return fmt.Errorf("template '%v' not found", name)
	}

	return template.ExecuteTemplate(w, name, data)
}

func New(ctx context.Context, bucketName string) (*echo.Echo, error) {
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	server := echo.New()

	parseTemplates(server)
	setupRoutes(server, gcsClient, bucketName)

	return server, nil
}

func parseTemplates(server *echo.Echo) {
	server.Renderer = &Templates{
		templates: map[string]*template.Template{
			"index.html": template.Must(template.New("").ParseFiles("templates/index.html")),
		},
	}
}

func setupRoutes(server *echo.Echo, gcsClient *storage.Client, bucket string) {
	server.GET("/", func(c echo.Context) error {
		dbtDocs := []string{}
		dbts := gcsClient.Bucket(bucket).Objects(c.Request().Context(), nil)
		for {
			o, err := dbts.Next()
			if errors.Is(err, iterator.Done) {
				break
			}

			if !contains(dbtDocs, strings.Split(o.Name, "/")[0]) {
				dbtDocs = append(dbtDocs, strings.Split(o.Name, "/")[0])
			}
		}

		return c.Render(http.StatusOK, "index.html", map[string]any{
			"dbtDocs": dbtDocs,
		})
	})

	server.GET("/:id", func(c echo.Context) error {
		dbtID := c.Param("id")
		return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/%v/index.html", dbtID))
	})

	server.GET("/:id/*", func(c echo.Context) error {
		objReader, err := gcsClient.Bucket(bucket).Object(strings.Split(strings.TrimLeft(c.Request().URL.String(), "/"), "?")[0]).NewReader(c.Request().Context())
		if err != nil {
			return nil
		}
		datab, err := io.ReadAll(objReader)
		if err != nil {
			return err
		}
		_, err = c.Response().Writer.Write(datab)
		return err
	})
}

func contains(files []string, file string) bool {
	for _, f := range files {
		if f == file {
			return true
		}
	}

	return false
}
