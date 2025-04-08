// internal/service/image_service.go
package service

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/nfnt/resize"

	"image-upload-server/internal/models"
	"image-upload-server/internal/repository"
)

// ImageService handles image processing and storage
type ImageService struct {
	repo *repository.S3Repository
}

// NewImageService creates a new image service
func NewImageService(repo *repository.S3Repository) *ImageService {
	return &ImageService{
		repo: repo,
	}
}

// ProcessAndUploadImage processes an image and uploads it to S3
func (s *ImageService) ProcessAndUploadImage(
	fileBytes []byte,
	filename string,
	compressSizes []models.CompressSpec,
) (*models.UploadResponse, error) {
	// Decode the image
	img, format, err := decodeImage(fileBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Generate a unique file name for the original image
	timestamp := time.Now().UnixNano()
	fileExt := strings.ToLower(filepath.Ext(filename))
	fileNameWithoutExt := strings.TrimSuffix(filename, fileExt)
	originalFileName := fmt.Sprintf("%s_%d%s", fileNameWithoutExt, timestamp, fileExt)

	// Upload original image to S3
	originalURL, err := s.repo.UploadFile(fileBytes, originalFileName, getContentType(format))
	if err != nil {
		return nil, fmt.Errorf("failed to upload original image: %w", err)
	}

	// Create response object
	originalBounds := img.Bounds()
	response := &models.UploadResponse{
		OriginalImage: models.ImageResult{
			Width:  originalBounds.Dx(),
			Height: originalBounds.Dy(),
			URL:    originalURL,
		},
		CompressedImages: []models.ImageResult{},
		Message:          "Image uploaded and processed successfully",
	}

	// Process and upload each compressed size
	for _, spec := range compressSizes {
		// Resize the image
		resizedImg := resize.Resize(uint(spec.Width), uint(spec.Height), img, resize.Lanczos3)

		// Encode the resized image
		var buf bytes.Buffer
		var encodeErr error

		if format == "jpeg" {
			encodeErr = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 85})
		} else {
			encodeErr = png.Encode(&buf, resizedImg)
		}

		if encodeErr != nil {
			log.Printf("Failed to encode compressed image: %v", encodeErr)
			continue
		}

		// Generate a unique filename for the compressed image
		compressedFileName := fmt.Sprintf("%s_%dx%d_%d%s",
			fileNameWithoutExt, spec.Width, spec.Height, timestamp, fileExt)

		// Upload the compressed image to S3
		compressedURL, uploadErr := s.repo.UploadFile(buf.Bytes(), compressedFileName, getContentType(format))
		if uploadErr != nil {
			log.Printf("Failed to upload compressed image: %v", uploadErr)
			continue
		}

		// Add to response
		response.CompressedImages = append(response.CompressedImages, models.ImageResult{
			Width:  spec.Width,
			Height: spec.Height,
			URL:    compressedURL,
		})
	}

	return response, nil
}

// GetImageInfo gets information about an image by filename
func (s *ImageService) GetImageInfo(filename string) (*models.ImageResult, error) {
	exists, err := s.repo.GetFile(filename)
	if err != nil || !exists {
		return nil, fmt.Errorf("image not found")
	}

	// Generate the URL for the image
	var imageURL string
	// Note: This requires access to the S3 config, which could be passed to the service
	// For now, we're using a simplified approach
	imageURL = fmt.Sprintf("https://s3-url/%s", filename)

	// Extract dimensions from filename if available (format: name_WxH_timestamp.ext)
	parts := strings.Split(filename, "_")
	if len(parts) >= 2 {
		dimParts := strings.Split(parts[len(parts)-2], "x")
		if len(dimParts) == 2 {
			width := 0
			height := 0
			fmt.Sscanf(dimParts[0], "%d", &width)
			fmt.Sscanf(dimParts[1], "%d", &height)
			if width > 0 && height > 0 {
				return &models.ImageResult{
					Width:  width,
					Height: height,
					URL:    imageURL,
				}, nil
			}
		}
	}

	// If dimensions can't be extracted, return just the URL
	return &models.ImageResult{
		Width:  0,
		Height: 0,
		URL:    imageURL,
	}, nil
}

// ListImages lists all images in the S3 bucket
func (s *ImageService) ListImages() ([]string, error) {
	return s.repo.ListFiles()
}

// Helper function to decode an image
func decodeImage(fileBytes []byte) (image.Image, string, error) {
	img, format, err := image.Decode(bytes.NewReader(fileBytes))
	return img, format, err
}

// Helper function to get content type from image format
func getContentType(format string) string {
	switch format {
	case "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	default:
		return "application/octet-stream"
	}
}
