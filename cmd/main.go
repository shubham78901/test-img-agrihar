// cmd/app/main.go
package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"image-upload-server/internal/config"
	"image-upload-server/internal/handlers"
	"image-upload-server/internal/repository"
	"image-upload-server/internal/service"

	// Import generated swagger docs
	_ "image-upload-server/docs"
)

// @title           Image Upload API
// @version         1.0
// @description     An API for uploading, compressing, and storing images to S3
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.example.com/support
// @contact.email  support@example.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

func main() {
	// Load configuration
	cfg := config.New()

	// Initialize repository
	s3Repo, err := repository.NewS3Repository(cfg.S3)
	if err != nil {
		log.Fatalf("Failed to initialize S3 repository: %v", err)
	}

	// Initialize service
	imgService := service.NewImageService(s3Repo)

	// Initialize handlers
	imgHandler := handlers.NewImageHandler(imgService)

	// Setup router
	r := setupRoutes(imgHandler)

	// Start server
	log.Printf("Server starting on port %s...", cfg.App.Port)
	log.Printf("Swagger documentation available at http://localhost:%s/swagger/index.html", cfg.App.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.App.Port, r))
}

func setupRoutes(h *handlers.ImageHandler) *mux.Router {
	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/upload", h.Upload).Methods("POST")
	api.HandleFunc("/images", h.ListImages).Methods("GET")
	api.HandleFunc("/images/{filename}", h.GetImage).Methods("GET")
	api.HandleFunc("/health", h.HealthCheck).Methods("GET")

	// Swagger documentation
	r.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The URL pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	return r
}
