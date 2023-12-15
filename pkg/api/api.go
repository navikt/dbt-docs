package api

import (
	"context"
	"net/http"

	"cloud.google.com/go/storage"
	"github.com/go-chi/chi"
)

func New(ctx context.Context) (*chi.Mux, error) {
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	setupRoutes(router, gcsClient)

	return router, nil
}

func setupRoutes(router *chi.Mux, gcsClient *storage.Client) {
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
	})

	// router.Route("/"+a.quartoPath, func(r chi.Router) {
	// 	r.Get("/*", a.GetDBTDoc)
	// })
}
