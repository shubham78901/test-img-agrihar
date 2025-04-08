// internal/handlers/image_handler.go
package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	"image-upload-server/internal/models"
	"image-upload-server/internal/service"
)

// ImageHandler handles HTTP requests for image operations
type ImageHandler struct {
	service *service.ImageService
}

// NewImageHandler creates a new image handler
func NewImageHandler(svc *service.ImageService) *ImageHandler {
	return &ImageHandler{
		service: svc,
	}
}

// Upload handles image upload requests
// @Summary Upload an image
// @Description Upload and compress an image based on specified sizes, then store in S3
// @Tags images
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Image to upload"
// @Param compress_sizes formData string true "JSON array of compression specifications [{'width': 100, 'height': 100}, ...]"
// @Success 200 {object} models.UploadResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /upload [post]
func (h *ImageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 32MB)
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to parse form: "+err.Error())
		return
	}

	// Get the file from the request
	file, header, err := r.FormFile("image")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to get image file: "+err.Error())
		return
	}
	defer file.Close()

	// Check file type
	fileExt := strings.ToLower(filepath.Ext(header.Filename))
	if fileExt != ".jpg" && fileExt != ".jpeg" && fileExt != ".png" {
		respondWithError(w, http.StatusBadRequest, "Unsupported file type. Only JPG and PNG are supported")
		return
	}

	// Read compress sizes from form data
	compressSizesStr := r.FormValue("compress_sizes")
	if compressSizesStr == "" {
		respondWithError(w, http.StatusBadRequest, "Missing compress_sizes parameter")
		return
	}

	var compressSizes []models.CompressSpec
	err = json.Unmarshal([]byte(compressSizesStr), &compressSizes)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid compress_sizes format: "+err.Error())
		return
	}

	// Read the entire file into memory
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to read file: "+err.Error())
		return
	}

	// Process and upload the image
	response, err := h.service.ProcessAndUploadImage(fileBytes, header.Filename, compressSizes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, response)
}

// GetImage handles image retrieval requests
// @Summary Get image information
// @Description Get information about an uploaded image by filename
// @Tags images
// @Produce json
// @Param filename path string true "Image filename"
// @Success 200 {object} models.ImageResult
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /images/{filename} [get]
func (h *ImageHandler) GetImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	filename := vars["filename"]

	// Get image info from service
	imageInfo, err := h.service.GetImageInfo(filename)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Image not found")
		return
	}

	respondWithJSON(w, http.StatusOK, imageInfo)
}

// ListImages handles image listing requests
// @Summary List all images
// @Description List all images in the S3 bucket
// @Tags images
// @Produce json
// @Success 200 {array} string
// @Failure 500 {object} models.ErrorResponse
// @Router /images [get]
func (h *ImageHandler) ListImages(w http.ResponseWriter, r *http.Request) {
	// Get image list from service
	images, err := h.service.ListImages()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to list images: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, images)
}

// HealthCheck handles health check requests
// @Summary Health check
// @Description Check if the API is running
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *ImageHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// Helper function to respond with JSON
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// Helper function to respond with an error
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, models.ErrorResponse{Error: message})
}
