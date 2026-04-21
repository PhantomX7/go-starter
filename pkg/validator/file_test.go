package validator

import (
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/go-playground/validator/v10"
)

// createMockFileHeader creates a mock multipart.FileHeader for testing
// This function creates a file header with specified filename and size for validation testing
func createMockFileHeader(filename string, size int64) *multipart.FileHeader {
	return &multipart.FileHeader{
		Filename: filename,
		Size:     size,
		Header: textproto.MIMEHeader{
			"Content-Disposition": []string{`form-data; name="file"; filename="` + filename + `"`},
			"Content-Type":        []string{"application/octet-stream"},
		},
	}
}

// setupFileValidator creates a validator instance with file validation functions registered
// This follows the same pattern as exist_test.go for consistent test setup
func setupFileValidator(t *testing.T) *validator.Validate {
	v := validator.New()

	// Create custom validator instance (no database needed for file validators)
	customValidator := New(nil)

	// Register file validation functions
	v.RegisterValidation("filesize", customValidator.FileSize())
	v.RegisterValidation("fileext", customValidator.FileExtension())

	return v
}

func TestFileSize_ValidatorFunction(t *testing.T) {
	v := setupFileValidator(t)

	// Test struct with file size validation
	type TestStruct struct {
		SmallFile    *multipart.FileHeader `validate:"filesize=1048576"`   // 1MB limit
		MediumFile   *multipart.FileHeader `validate:"filesize=5242880"`   // 5MB limit
		LargeFile    *multipart.FileHeader `validate:"filesize=10485760"`  // 10MB limit
		DefaultFile  *multipart.FileHeader `validate:"filesize"`           // Default 10MB limit
		OptionalFile *multipart.FileHeader `validate:"omitempty,filesize"` // Optional with default limit
	}

	tests := []struct {
		name        string
		input       TestStruct
		expectValid bool
		description string
	}{
		{
			name: "valid file sizes within limits",
			input: TestStruct{
				SmallFile:    createMockFileHeader("small.txt", 512000),    // 500KB < 1MB
				MediumFile:   createMockFileHeader("medium.pdf", 3145728),  // 3MB < 5MB
				LargeFile:    createMockFileHeader("large.zip", 8388608),   // 8MB < 10MB
				DefaultFile:  createMockFileHeader("default.doc", 5242880), // 5MB < 10MB default
				OptionalFile: nil,
			},
			expectValid: true,
			description: "Should be valid with file sizes within their respective limits",
		},
		{
			name: "valid file sizes at exact limits",
			input: TestStruct{
				SmallFile:    createMockFileHeader("small.txt", 1048576),    // Exactly 1MB
				MediumFile:   createMockFileHeader("medium.pdf", 5242880),   // Exactly 5MB
				LargeFile:    createMockFileHeader("large.zip", 10485760),   // Exactly 10MB
				DefaultFile:  createMockFileHeader("default.doc", 10485760), // Exactly 10MB default
				OptionalFile: nil,
			},
			expectValid: true,
			description: "Should be valid with file sizes at exact limits",
		},
		{
			name: "invalid oversized small file",
			input: TestStruct{
				SmallFile:    createMockFileHeader("small.txt", 1048577),   // 1MB + 1 byte
				MediumFile:   createMockFileHeader("medium.pdf", 3145728),  // 3MB < 5MB
				LargeFile:    createMockFileHeader("large.zip", 8388608),   // 8MB < 10MB
				DefaultFile:  createMockFileHeader("default.doc", 5242880), // 5MB < 10MB default
				OptionalFile: nil,
			},
			expectValid: false,
			description: "Should be invalid when small file exceeds 1MB limit",
		},
		{
			name: "invalid oversized medium file",
			input: TestStruct{
				SmallFile:    createMockFileHeader("small.txt", 512000),    // 500KB < 1MB
				MediumFile:   createMockFileHeader("medium.pdf", 5242881),  // 5MB + 1 byte
				LargeFile:    createMockFileHeader("large.zip", 8388608),   // 8MB < 10MB
				DefaultFile:  createMockFileHeader("default.doc", 5242880), // 5MB < 10MB default
				OptionalFile: nil,
			},
			expectValid: false,
			description: "Should be invalid when medium file exceeds 5MB limit",
		},
		{
			name: "invalid oversized large file",
			input: TestStruct{
				SmallFile:    createMockFileHeader("small.txt", 512000),    // 500KB < 1MB
				MediumFile:   createMockFileHeader("medium.pdf", 3145728),  // 3MB < 5MB
				LargeFile:    createMockFileHeader("large.zip", 10485761),  // 10MB + 1 byte
				DefaultFile:  createMockFileHeader("default.doc", 5242880), // 5MB < 10MB default
				OptionalFile: nil,
			},
			expectValid: false,
			description: "Should be invalid when large file exceeds 10MB limit",
		},
		{
			name: "invalid oversized default file",
			input: TestStruct{
				SmallFile:    createMockFileHeader("small.txt", 512000),     // 500KB < 1MB
				MediumFile:   createMockFileHeader("medium.pdf", 3145728),   // 3MB < 5MB
				LargeFile:    createMockFileHeader("large.zip", 8388608),    // 8MB < 10MB
				DefaultFile:  createMockFileHeader("default.doc", 10485761), // 10MB + 1 byte (exceeds default)
				OptionalFile: nil,
			},
			expectValid: false,
			description: "Should be invalid when default file exceeds 10MB default limit",
		},
		{
			name: "invalid empty files",
			input: TestStruct{
				SmallFile:    createMockFileHeader("small.txt", 0),         // Empty file
				MediumFile:   createMockFileHeader("medium.pdf", 3145728),  // 3MB < 5MB
				LargeFile:    createMockFileHeader("large.zip", 8388608),   // 8MB < 10MB
				DefaultFile:  createMockFileHeader("default.doc", 5242880), // 5MB < 10MB default
				OptionalFile: nil,
			},
			expectValid: false,
			description: "Should be invalid with empty files (size = 0)",
		},
		{
			name: "valid with optional file present",
			input: TestStruct{
				SmallFile:    createMockFileHeader("small.txt", 512000),     // 500KB < 1MB
				MediumFile:   createMockFileHeader("medium.pdf", 3145728),   // 3MB < 5MB
				LargeFile:    createMockFileHeader("large.zip", 8388608),    // 8MB < 10MB
				DefaultFile:  createMockFileHeader("default.doc", 5242880),  // 5MB < 10MB default
				OptionalFile: createMockFileHeader("optional.txt", 1048576), // 1MB < 10MB default
			},
			expectValid: true,
			description: "Should be valid with optional file present and within limits",
		},
		{
			name: "invalid with oversized optional file",
			input: TestStruct{
				SmallFile:    createMockFileHeader("small.txt", 512000),      // 500KB < 1MB
				MediumFile:   createMockFileHeader("medium.pdf", 3145728),    // 3MB < 5MB
				LargeFile:    createMockFileHeader("large.zip", 8388608),     // 8MB < 10MB
				DefaultFile:  createMockFileHeader("default.doc", 5242880),   // 5MB < 10MB default
				OptionalFile: createMockFileHeader("optional.txt", 10485761), // 10MB + 1 byte (exceeds default)
			},
			expectValid: false,
			description: "Should be invalid when optional file exceeds default limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(tt.input)
			isValid := err == nil

			if isValid != tt.expectValid {
				t.Errorf("Validation result = %v, want %v. %s", isValid, tt.expectValid, tt.description)
				if err != nil {
					t.Errorf("Validation error: %v", err)
				}
			}
		})
	}
}

func TestFileExtension_ValidatorFunction(t *testing.T) {
	v := setupFileValidator(t)

	// Test struct with file extension validation
	type TestStruct struct {
		ImageFile    *multipart.FileHeader `validate:"fileext=jpg&png&gif"`
		DocumentFile *multipart.FileHeader `validate:"fileext=pdf&doc&docx"`
		TextFile     *multipart.FileHeader `validate:"fileext=txt&md"`
		AnyImageFile *multipart.FileHeader `validate:"fileext=jpg&jpeg&png&gif&bmp&webp"`
		OptionalFile *multipart.FileHeader `validate:"omitempty,fileext=pdf&txt"`
	}

	tests := []struct {
		name        string
		input       TestStruct
		expectValid bool
		description string
	}{
		{
			name: "valid file extensions",
			input: TestStruct{
				ImageFile:    createMockFileHeader("photo.jpg", 1024),
				DocumentFile: createMockFileHeader("document.pdf", 2048),
				TextFile:     createMockFileHeader("readme.txt", 512),
				AnyImageFile: createMockFileHeader("image.png", 1536),
				OptionalFile: nil,
			},
			expectValid: true,
			description: "Should be valid with correct file extensions",
		},
		{
			name: "valid case insensitive extensions",
			input: TestStruct{
				ImageFile:    createMockFileHeader("photo.JPG", 1024),    // Uppercase
				DocumentFile: createMockFileHeader("document.PDF", 2048), // Uppercase
				TextFile:     createMockFileHeader("readme.TXT", 512),    // Uppercase
				AnyImageFile: createMockFileHeader("image.PNG", 1536),    // Uppercase
				OptionalFile: nil,
			},
			expectValid: true,
			description: "Should be valid with case insensitive extensions",
		},
		{
			name: "valid mixed case extensions",
			input: TestStruct{
				ImageFile:    createMockFileHeader("photo.Jpg", 1024),    // Mixed case
				DocumentFile: createMockFileHeader("document.Pdf", 2048), // Mixed case
				TextFile:     createMockFileHeader("readme.Txt", 512),    // Mixed case
				AnyImageFile: createMockFileHeader("image.Png", 1536),    // Mixed case
				OptionalFile: nil,
			},
			expectValid: true,
			description: "Should be valid with mixed case extensions",
		},
		{
			name: "valid alternative extensions",
			input: TestStruct{
				ImageFile:    createMockFileHeader("photo.gif", 1024),
				DocumentFile: createMockFileHeader("document.docx", 2048),
				TextFile:     createMockFileHeader("readme.md", 512),
				AnyImageFile: createMockFileHeader("image.webp", 1536),
				OptionalFile: nil,
			},
			expectValid: true,
			description: "Should be valid with alternative allowed extensions",
		},
		{
			name: "invalid image file extension",
			input: TestStruct{
				ImageFile:    createMockFileHeader("photo.bmp", 1024), // bmp not allowed for ImageFile
				DocumentFile: createMockFileHeader("document.pdf", 2048),
				TextFile:     createMockFileHeader("readme.txt", 512),
				AnyImageFile: createMockFileHeader("image.png", 1536),
				OptionalFile: nil,
			},
			expectValid: false,
			description: "Should be invalid with disallowed image extension",
		},
		{
			name: "invalid document file extension",
			input: TestStruct{
				ImageFile:    createMockFileHeader("photo.jpg", 1024),
				DocumentFile: createMockFileHeader("document.txt", 2048), // txt not allowed for DocumentFile
				TextFile:     createMockFileHeader("readme.txt", 512),
				AnyImageFile: createMockFileHeader("image.png", 1536),
				OptionalFile: nil,
			},
			expectValid: false,
			description: "Should be invalid with disallowed document extension",
		},
		{
			name: "invalid text file extension",
			input: TestStruct{
				ImageFile:    createMockFileHeader("photo.jpg", 1024),
				DocumentFile: createMockFileHeader("document.pdf", 2048),
				TextFile:     createMockFileHeader("readme.pdf", 512), // pdf not allowed for TextFile
				AnyImageFile: createMockFileHeader("image.png", 1536),
				OptionalFile: nil,
			},
			expectValid: false,
			description: "Should be invalid with disallowed text extension",
		},
		{
			name: "invalid files without extensions",
			input: TestStruct{
				ImageFile:    createMockFileHeader("photo", 1024), // No extension
				DocumentFile: createMockFileHeader("document.pdf", 2048),
				TextFile:     createMockFileHeader("readme.txt", 512),
				AnyImageFile: createMockFileHeader("image.png", 1536),
				OptionalFile: nil,
			},
			expectValid: false,
			description: "Should be invalid with files that have no extension",
		},
		{
			name: "invalid files with dot but no extension",
			input: TestStruct{
				ImageFile:    createMockFileHeader("photo.", 1024), // Dot but no extension
				DocumentFile: createMockFileHeader("document.pdf", 2048),
				TextFile:     createMockFileHeader("readme.txt", 512),
				AnyImageFile: createMockFileHeader("image.png", 1536),
				OptionalFile: nil,
			},
			expectValid: false,
			description: "Should be invalid with files that have dot but no extension",
		},
		{
			name: "valid with optional file present",
			input: TestStruct{
				ImageFile:    createMockFileHeader("photo.jpg", 1024),
				DocumentFile: createMockFileHeader("document.pdf", 2048),
				TextFile:     createMockFileHeader("readme.txt", 512),
				AnyImageFile: createMockFileHeader("image.png", 1536),
				OptionalFile: createMockFileHeader("optional.pdf", 1024), // pdf allowed for OptionalFile
			},
			expectValid: true,
			description: "Should be valid with optional file present and correct extension",
		},
		{
			name: "invalid with wrong optional file extension",
			input: TestStruct{
				ImageFile:    createMockFileHeader("photo.jpg", 1024),
				DocumentFile: createMockFileHeader("document.pdf", 2048),
				TextFile:     createMockFileHeader("readme.txt", 512),
				AnyImageFile: createMockFileHeader("image.png", 1536),
				OptionalFile: createMockFileHeader("optional.doc", 1024), // doc not allowed for OptionalFile
			},
			expectValid: false,
			description: "Should be invalid when optional file has wrong extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(tt.input)
			isValid := err == nil

			if isValid != tt.expectValid {
				t.Errorf("Validation result = %v, want %v. %s", isValid, tt.expectValid, tt.description)
				if err != nil {
					t.Errorf("Validation error: %v", err)
				}
			}
		})
	}
}
