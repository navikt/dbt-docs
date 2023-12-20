package api

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/navikt/dbt-docs/pkg/bigquery"
	"github.com/navikt/dbt-docs/pkg/gcs"
)

const (
	maxMemoryMultipartForm = 32 << 20
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

func New(ctx context.Context, bucket string, bq *bigquery.BigQuery, logger *slog.Logger) (*echo.Echo, error) {
	gcs, err := gcs.New(ctx, bucket)
	if err != nil {
		return nil, err
	}

	server := echo.New()
	server.Pre(middleware.RemoveTrailingSlash())
	parseTemplates(server)
	setupRoutes(server, gcs, bq, logger)

	return server, nil
}

func parseTemplates(server *echo.Echo) {
	server.Renderer = &Templates{
		templates: map[string]*template.Template{
			"index.html": template.Must(template.New("").ParseFiles("templates/index.html")),
		},
	}
}

func setupRoutes(server *echo.Echo, gcs *gcs.GCSClient, bq *bigquery.BigQuery, logger *slog.Logger) {
	server.GET("/", func(c echo.Context) error {
		teamsDocsMap := gcs.ListTeamsAndDocsInBucket(c.Request().Context())
		return c.Render(http.StatusOK, "index.html", map[string]any{
			"dbtDocs": teamsDocsMap,
		})
	})

	server.GET("/docs", func(c echo.Context) error {
		return c.Redirect(http.StatusSeeOther, "/")
	})

	server.GET("/docs/:team/:id", func(c echo.Context) error {
		teamID := c.Param("team")
		docID := c.Param("id")
		return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/docs/%v/%v/index.html", teamID, docID))
	})

	server.GET("/docs/:team/:id/*", func(c echo.Context) error {
		bucketFilePath := bucketFilePathFromURLPath(c.Request().URL.String())
		objectBytes, err := gcs.GetFile(c.Request().Context(), bucketFilePath)
		if err != nil {
			logger.Error(fmt.Sprintf("unable to read object %v, error: %v", bucketFilePath, err.Error()), "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"status":  "error",
				"message": fmt.Sprintf("unable to read object '%v'", bucketFilePath),
			})
		}

		if url, ok := isIndexPage(c); ok {
			if err := bq.RegisterView(c.Request().Context(), url); err != nil {
				logger.Error("registering dbt doc view in bigquery", "error", err)
			}
		}

		_, err = c.Response().Writer.Write(objectBytes)
		return err
	})

	server.POST("/docs/:team/:id", func(c echo.Context) error {
		teamID := c.Param("team")
		docID := c.Param("id")

		docContent := gcs.ListFilesWithPrefix(c.Request().Context(), fmt.Sprintf("docs/%v/%v", teamID, docID))
		if len(docContent) > 0 {
			return c.JSON(http.StatusConflict, map[string]string{
				"status":  "error",
				"message": fmt.Sprintf("dbt doc with id '%v' already exists", docID),
			})
		}

		if err := uploadDocs(c, teamID, docID, gcs); err != nil {
			logger.Error(fmt.Sprintf("unable to upload dbt doc %v to bucket", docID), "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"status":  "error",
				"message": fmt.Sprintf("unable to upload dbt doc %v to bucket", docID),
			})
		}
		return c.JSON(http.StatusCreated, map[string]string{
			"status":  "created",
			"message": fmt.Sprintf("created dbt docs for '%v'", docID),
		})
	})

	server.PUT("/docs/:team/:id", func(c echo.Context) error {
		teamID := c.Param("team")
		docID := c.Param("id")

		if err := gcs.DeleteFilesWithPrefix(c.Request().Context(), fmt.Sprintf("docs/%v/%v", teamID, docID)); err != nil {
			logger.Error(fmt.Sprintf("error deleting dbt doc '%v' before updating", docID), "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"status":  "error",
				"message": fmt.Sprintf("error deleting dbt doc '%v' before updating", docID),
			})
		}

		if err := uploadDocs(c, teamID, docID, gcs); err != nil {
			logger.Error(fmt.Sprintf("error uploading dbt doc '%v'", docID), "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"status":  "error",
				"message": fmt.Sprintf("error uploading dbt doc '%v' to bucket", docID),
			})
		}
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "updated",
			"message": fmt.Sprintf("updated dbt docs for %v", docID),
		})
	})

	server.PATCH("/docs/:team/:id", func(c echo.Context) error {
		teamID := c.Param("team")
		docID := c.Param("id")
		if err := uploadDocs(c, teamID, docID, gcs); err != nil {
			logger.Error(fmt.Sprintf("error uploading dbt doc '%v'", docID), "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"status":  "error",
				"message": fmt.Sprintf("error uploading dbt doc '%v' to bucket", docID),
			})
		}
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "updated",
			"message": fmt.Sprintf("updated dbt docs for %v", docID),
		})
	})
}

func bucketFilePathFromURLPath(urlPath string) string {
	withoutPathPrefix := strings.TrimPrefix(urlPath, "/")
	return strings.Split(withoutPathPrefix, "?")[0]
}

func uploadDocs(c echo.Context, teamID, docID string, gcs *gcs.GCSClient) error {
	if err := c.Request().ParseMultipartForm(maxMemoryMultipartForm); err != nil {
		return err
	}

	for fileName, fileHeader := range c.Request().MultipartForm.File {
		file, err := fileHeader[0].Open()
		if err != nil {
			return err
		}
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			return err
		}

		if fileName == "index.html" {
			fileBytes = addHomeLink(fileBytes)
		}

		if err := gcs.UploadFile(c.Request().Context(), fmt.Sprintf("docs/%v/%v/%v", teamID, docID, fileName), fileBytes); err != nil {
			return err
		}
	}
	return nil
}

func isIndexPage(c echo.Context) (string, bool) {
	urlPathParts := strings.Split(c.Request().URL.Path, "/")
	if urlPathParts[len(urlPathParts)-1] == "index.html" {
		path := strings.Join(urlPathParts[:len(urlPathParts)-1], "")
		return fmt.Sprintf("%v://%v/%v", c.Scheme(), c.Request().Host, path), true
	}
	return "", false
}

func addHomeLink(fileBytes []byte) []byte {
	r, _ := regexp.Compile(`<img.*"{{ logo }}" ?\/>`)
	logoElement := r.FindString(string(fileBytes))
	altered := strings.Replace(
		string(fileBytes),
		logoElement,
		fmt.Sprintf(`<a href="/">%v</a>`, logoElement),
		1,
	)

	return []byte(altered)
}
