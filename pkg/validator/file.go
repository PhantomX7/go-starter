// Package validator provides custom file validation functions for the go-playground/validator library.
//
// This package extends the standard validator with specialized file validation capabilities,
// focusing on security, performance, and reliability for file upload operations.
//
// The validators in this package are designed to work with multipart.FileHeader objects
// from HTTP file uploads and provide comprehensive validation for:
//   - File size limits and constraints
//   - File extension validation and filtering
//   - MIME type detection and validation
//
// Security Features:
//   - Content-based MIME type detection to prevent file spoofing
//   - File size limits to prevent DoS attacks
//   - Extension filtering to restrict file types
//   - Automatic resource cleanup and error handling
//
// Parameter Separator Usage:
//
//	For validators that accept multiple values (FileExtension and FileMimeType),
//	parameters must be separated using ampersand (&) character.
//
// Usage:
//
//	These validators are typically used in HTTP handlers that process file uploads,
//	providing server-side validation before file processing or storage.
//
// Example Integration:
//
//	validator := validator.New()
//	cv := customValidator{validator: validator, db: db}
//	validator.RegisterValidation("filesize", cv.FileSize())
//	validator.RegisterValidation("fileext", cv.FileExtension())
//	validator.RegisterValidation("filemime", cv.FileMimeType())
package validator

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
)

// FileSize validates the size of an uploaded file against a maximum size limit.
//
// This validator ensures that uploaded files do not exceed the specified size limit,
// preventing potential DoS attacks and managing server storage efficiently.
//
// Parameters:
//   - The validator accepts an optional parameter specifying the maximum file size in bytes.
//   - If no parameter is provided, the default maximum size is 10MB (10 << 20 bytes).
//   - Parameter format: "1048576" (for 1MB limit)
//
// Validation Rules:
//   - File size must be greater than 0 bytes (non-empty file)
//   - File size must be less than or equal to the specified maximum size
//   - The field must be of type multipart.FileHeader
//
// Usage Examples:
//   type FileUploadRequest struct {
//       Avatar *multipart.FileHeader `validate:"filesize=2097152"` // 2MB limit
//       Document *multipart.FileHeader `validate:"filesize"`        // 10MB default limit
//   }
//
// Returns:
//   - true if the file size is valid (0 < size <= maxSize)
//   - false if the file is invalid, empty, or exceeds the size limit
func (cv customValidator) FileSize() validator.Func {
	return func(fl validator.FieldLevel) bool {
		file, ok := fl.Field().Interface().(multipart.FileHeader)
		if !ok {
			return false
		}

		maxSize := int64(10 << 20) // 10MB default

		// You can get the max size from tag parameter
		if fl.Param() != "" {
			fmt.Sscanf(fl.Param(), "%d", &maxSize)
		}

		return file.Size <= maxSize && file.Size > 0
	}
}

// FileExtension validates that an uploaded file has one of the allowed file extensions.
//
// This validator provides security by restricting file uploads to specific file types,
// helping prevent malicious file uploads and ensuring only expected file formats are accepted.
//
// Parameters:
//   - Required parameter specifying allowed file extensions separated by ampersand (&) characters
//   - Extensions should be specified without the leading dot
//   - Parameter format: "jpg&png&gif" or "pdf&doc&docx"
//   - Case-insensitive validation (both .JPG and .jpg are accepted)
//
// Validation Rules:
//   - The field must be of type multipart.FileHeader
//   - File must have a valid filename with an extension
//   - File extension must match one of the allowed extensions (case-insensitive)
//   - Extensions are compared after converting to lowercase
//
// Usage Examples:
//   type FileUploadRequest struct {
//       Image *multipart.FileHeader `validate:"fileext=jpg&png&gif"`
//       Document *multipart.FileHeader `validate:"fileext=pdf&doc&docx"`
//       Avatar *multipart.FileHeader `validate:"fileext=jpg&jpeg&png"`
//   }
//
// Security Considerations:
//   - This validation only checks file extensions, not actual file content
//   - Should be combined with MIME type validation for enhanced security
//   - Consider using FileMimeType validator for content-based validation
//
// Returns:
//   - true if the file extension is in the allowed list
//   - false if the file extension is not allowed or file is invalid
func (cv customValidator) FileExtension() validator.Func {
	return func(fl validator.FieldLevel) bool {
		file, ok := fl.Field().Interface().(multipart.FileHeader)
		if !ok {
			return false
		}

		param := fl.Param()

		allowedExts := strings.Split(param, "&")
		
		ext := strings.ToLower(filepath.Ext(file.Filename))
		
		// If no extension, return false
		if ext == "" {
			return false
		}
		
		// Remove the dot from extension for comparison
		ext = ext[1:]

		for _, allowed := range allowedExts {
			// Trim whitespace and convert to lowercase for comparison
			allowed = strings.ToLower(strings.TrimSpace(allowed))
			if ext == allowed {
				return true
			}
		}
		return false
	}
}
