package api

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
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

func New(ctx context.Context, bucket string, logger *slog.Logger) (*echo.Echo, error) {
	gcs, err := gcs.New(ctx, bucket)
	if err != nil {
		return nil, err
	}

	server := echo.New()
	parseTemplates(server)
	setupRoutes(server, gcs, logger)

	return server, nil
}

func parseTemplates(server *echo.Echo) {
	server.Renderer = &Templates{
		templates: map[string]*template.Template{
			"index.html": template.Must(template.New("").ParseFiles("templates/index.html")),
		},
	}
}

func setupRoutes(server *echo.Echo, gcs *gcs.GCSClient, logger *slog.Logger) {
	server.GET("/", func(c echo.Context) error {
		dbtDocs := gcs.ListBucketRootFolders(c.Request().Context())
		return c.Render(http.StatusOK, "index.html", map[string]any{
			"dbtDocs": dbtDocs,
		})
	})

	server.POST("/docs/:id", func(c echo.Context) error {
		docID := c.Param("id")

		docContent := gcs.ListFilesWithPrefix(c.Request().Context(), docID)
		if len(docContent) > 0 {
			return c.JSON(http.StatusConflict, map[string]string{
				"status":  "error",
				"message": fmt.Sprintf("dbt doc with id '%v' already exists", docID),
			})
		}

		if err := uploadDocs(c, docID, gcs); err != nil {
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

	server.PUT("/docs/:id", func(c echo.Context) error {
		docID := c.Param("id")

		if err := gcs.DeleteFilesWithPrefix(c.Request().Context(), docID); err != nil {
			logger.Error(fmt.Sprintf("error deleting dbt doc '%v' before updating", docID), "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"status":  "error",
				"message": fmt.Sprintf("error deleting dbt doc '%v' before updating", docID),
			})
		}

		if err := uploadDocs(c, docID, gcs); err != nil {
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

	server.PATCH("/docs/:id", func(c echo.Context) error {
		docID := c.Param("id")
		if err := uploadDocs(c, docID, gcs); err != nil {
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

	server.GET("/docs/:id", func(c echo.Context) error {
		dbtID := c.Param("id")
		return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/docs/%v/index.html", dbtID))
	})

	server.GET("/docs/:id/*", func(c echo.Context) error {
		bucketFilePath := bucketFilePathFromURLPath(c.Request().URL.String())
		objectBytes, err := gcs.GetFile(c.Request().Context(), bucketFilePath)
		if err != nil {
			logger.Error(fmt.Sprintf("unable to read object, error: %v", err.Error()), "error", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"status":  "error",
				"message": fmt.Sprintf("unable to read object '%v'", bucketFilePath),
			})
		}
		_, err = c.Response().Writer.Write(objectBytes)
		return err
	})
}

func bucketFilePathFromURLPath(urlPath string) string {
	withoutPathPrefix := strings.TrimPrefix(urlPath, "/docs/")
	return strings.Split(withoutPathPrefix, "?")[0]
}

func uploadDocs(c echo.Context, docID string, gcs *gcs.GCSClient) error {
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
		if err := gcs.UploadFile(c.Request().Context(), fmt.Sprintf("%v/%v", docID, fileName), fileBytes); err != nil {
			return err
		}
	}
	return nil
}
