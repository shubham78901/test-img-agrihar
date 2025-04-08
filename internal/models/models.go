// internal/models/models.go
package models

// CompressSpec defines a compression specification for an image
type CompressSpec struct {
	Width  int `json:"width" example:"800"`  // Width in pixels
	Height int `json:"height" example:"600"` // Height in pixels
}

// ImageResult contains information about a processed image
type ImageResult struct {
	Width  int    `json:"width" example:"1920"`                                          // Width in pixels
	Height int    `json:"height" example:"1080"`                                         // Height in pixels
	URL    string `json:"url" example:"https://bucket.s3.region.amazonaws.com/file.jpg"` // S3 URL of the image
}

// UploadResponse is the response for a successful upload
type UploadResponse struct {
	OriginalImage    ImageResult   `json:"original_image"`                                              // Information about the original image
	CompressedImages []ImageResult `json:"compressed_images"`                                           // Information about all compressed versions
	Message          string        `json:"message" example:"Image uploaded and processed successfully"` // Status message
}

// ErrorResponse is the response for an error
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid file format"` // Error message
}
