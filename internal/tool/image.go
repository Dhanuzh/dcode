package tool

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ImageTool provides image analysis capabilities using vision models
func ImageTool() *ToolDef {
	return &ToolDef{
		Name:        "Image",
		Description: "Analyze images: describe, OCR, detect objects, answer questions.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"description": "Image operation to perform",
					"enum": []string{
						"describe", // Describe image contents
						"ocr",      // Extract text from image
						"question", // Answer question about image
						"compare",  // Compare two images
						"encode",   // Encode image to base64
						"info",     // Get image metadata
					},
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the image file",
				},
				"paths": map[string]interface{}{
					"type":        "array",
					"description": "Paths to multiple images (for compare)",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"question": map[string]interface{}{
					"type":        "string",
					"description": "Question to ask about the image",
				},
				"detail": map[string]interface{}{
					"type":        "string",
					"description": "Detail level for vision analysis (low, medium, high)",
					"enum":        []string{"low", "medium", "high"},
				},
			},
			"required": []string{"operation"},
		},
		Execute: executeImage,
	}
}

func executeImage(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
	operation, ok := input["operation"].(string)
	if !ok {
		return &ToolResult{
			Output:  "operation parameter is required",
			IsError: true,
		}, nil
	}

	switch operation {
	case "describe":
		path, ok := input["path"].(string)
		if !ok {
			return &ToolResult{
				Output:  "path parameter is required for describe",
				IsError: true,
			}, nil
		}
		return describeImage(ctx, tc, path, input)

	case "ocr":
		path, ok := input["path"].(string)
		if !ok {
			return &ToolResult{
				Output:  "path parameter is required for ocr",
				IsError: true,
			}, nil
		}
		return extractTextFromImage(ctx, tc, path)

	case "question":
		path, ok := input["path"].(string)
		if !ok {
			return &ToolResult{
				Output:  "path parameter is required for question",
				IsError: true,
			}, nil
		}
		question, ok := input["question"].(string)
		if !ok {
			return &ToolResult{
				Output:  "question parameter is required for question operation",
				IsError: true,
			}, nil
		}
		return answerImageQuestion(ctx, tc, path, question)

	case "compare":
		pathsInterface, ok := input["paths"].([]interface{})
		if !ok {
			return &ToolResult{
				Output:  "paths parameter (array) is required for compare",
				IsError: true,
			}, nil
		}
		paths := interfaceSliceToStringSlice(pathsInterface)
		if len(paths) < 2 {
			return &ToolResult{
				Output:  "at least 2 image paths are required for compare",
				IsError: true,
			}, nil
		}
		return compareImages(ctx, tc, paths)

	case "encode":
		path, ok := input["path"].(string)
		if !ok {
			return &ToolResult{
				Output:  "path parameter is required for encode",
				IsError: true,
			}, nil
		}
		return encodeImage(tc, path)

	case "info":
		path, ok := input["path"].(string)
		if !ok {
			return &ToolResult{
				Output:  "path parameter is required for info",
				IsError: true,
			}, nil
		}
		return getImageInfo(tc, path)

	default:
		return &ToolResult{
			Output:  fmt.Sprintf("unknown image operation: %s", operation),
			IsError: true,
		}, nil
	}
}

// describeImage generates a description of an image
func describeImage(ctx context.Context, tc *ToolContext, path string, input map[string]interface{}) (*ToolResult, error) {
	// Resolve path
	fullPath := resolvePath(tc.WorkDir, path)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return &ToolResult{
			Output:  fmt.Sprintf("image file not found: %s", path),
			IsError: true,
		}, nil
	}

	// Read and encode image
	imageData, err := os.ReadFile(fullPath)
	if err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("failed to read image: %v", err),
			IsError: true,
		}, nil
	}

	encoded := base64.StdEncoding.EncodeToString(imageData)

	// Note: In a real implementation, this would call a vision model API
	// For now, return information about the image and instructions
	output := fmt.Sprintf("# Image Analysis: %s\n\n", filepath.Base(path))
	output += fmt.Sprintf("**File:** %s\n", fullPath)
	output += fmt.Sprintf("**Size:** %d bytes\n", len(imageData))
	output += fmt.Sprintf("**Base64 Length:** %d characters\n\n", len(encoded))
	output += "**Note:** This tool prepares images for vision model analysis.\n\n"
	output += "To analyze this image, you would:\n"
	output += "1. Use a vision-capable model (e.g., GPT-4 Vision, Claude 3, Gemini Pro Vision)\n"
	output += "2. Send the base64-encoded image to the model\n"
	output += "3. Request description, OCR, or other analysis\n\n"
	output += "**Integration:** This requires integrating with provider vision APIs.\n"

	return &ToolResult{
		Output:  output,
		IsError: false,
	}, nil
}

// extractTextFromImage performs OCR on an image
func extractTextFromImage(ctx context.Context, tc *ToolContext, path string) (*ToolResult, error) {
	fullPath := resolvePath(tc.WorkDir, path)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return &ToolResult{
			Output:  fmt.Sprintf("image file not found: %s", path),
			IsError: true,
		}, nil
	}

	output := fmt.Sprintf("# OCR: %s\n\n", filepath.Base(path))
	output += "**Note:** OCR requires integration with:\n"
	output += "- Vision models (GPT-4 Vision, Claude 3, Gemini Pro Vision)\n"
	output += "- Dedicated OCR services (Tesseract, Google Cloud Vision, AWS Textract)\n\n"
	output += "**Recommended approach:**\n"
	output += "1. Use vision model with OCR-specific prompt\n"
	output += "2. Or integrate Tesseract: `tesseract image.png stdout`\n"

	return &ToolResult{
		Output:  output,
		IsError: false,
	}, nil
}

// answerImageQuestion answers a question about an image
func answerImageQuestion(ctx context.Context, tc *ToolContext, path, question string) (*ToolResult, error) {
	fullPath := resolvePath(tc.WorkDir, path)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return &ToolResult{
			Output:  fmt.Sprintf("image file not found: %s", path),
			IsError: true,
		}, nil
	}

	output := fmt.Sprintf("# Image Q&A: %s\n\n", filepath.Base(path))
	output += fmt.Sprintf("**Question:** %s\n\n", question)
	output += "**Note:** Answering questions about images requires vision model integration.\n\n"
	output += "**Implementation needed:**\n"
	output += "1. Encode image to base64\n"
	output += "2. Send to vision model with question\n"
	output += "3. Return model's answer\n"

	return &ToolResult{
		Output:  output,
		IsError: false,
	}, nil
}

// compareImages compares multiple images
func compareImages(ctx context.Context, tc *ToolContext, paths []string) (*ToolResult, error) {
	output := "# Image Comparison\n\n"
	output += fmt.Sprintf("**Images:** %d\n\n", len(paths))

	for i, path := range paths {
		fullPath := resolvePath(tc.WorkDir, path)
		if stat, err := os.Stat(fullPath); err == nil {
			output += fmt.Sprintf("%d. %s (%d bytes)\n", i+1, filepath.Base(path), stat.Size())
		} else {
			output += fmt.Sprintf("%d. %s (not found)\n", i+1, filepath.Base(path))
		}
	}

	output += "\n**Note:** Image comparison requires vision model integration.\n"

	return &ToolResult{
		Output:  output,
		IsError: false,
	}, nil
}

// encodeImage encodes an image to base64
func encodeImage(tc *ToolContext, path string) (*ToolResult, error) {
	fullPath := resolvePath(tc.WorkDir, path)

	imageData, err := os.ReadFile(fullPath)
	if err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("failed to read image: %v", err),
			IsError: true,
		}, nil
	}

	// Detect MIME type from extension
	ext := strings.ToLower(filepath.Ext(path))
	mimeType := "image/jpeg"
	switch ext {
	case ".png":
		mimeType = "image/png"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	case ".bmp":
		mimeType = "image/bmp"
	}

	encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

	output := fmt.Sprintf("# Base64 Encoded: %s\n\n", filepath.Base(path))
	output += fmt.Sprintf("**MIME Type:** %s\n", mimeType)
	output += fmt.Sprintf("**Original Size:** %d bytes\n", len(imageData))
	output += fmt.Sprintf("**Encoded Length:** %d characters\n\n", len(encoded))
	output += "**Data URI (truncated):**\n"
	if len(dataURI) > 200 {
		output += fmt.Sprintf("```\n%s...\n```\n", dataURI[:200])
	} else {
		output += fmt.Sprintf("```\n%s\n```\n", dataURI)
	}

	return &ToolResult{
		Output:  output,
		IsError: false,
	}, nil
}

// getImageInfo gets metadata about an image
func getImageInfo(tc *ToolContext, path string) (*ToolResult, error) {
	fullPath := resolvePath(tc.WorkDir, path)

	stat, err := os.Stat(fullPath)
	if err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("failed to stat image: %v", err),
			IsError: true,
		}, nil
	}

	output := fmt.Sprintf("# Image Info: %s\n\n", filepath.Base(path))
	output += fmt.Sprintf("**Full Path:** %s\n", fullPath)
	output += fmt.Sprintf("**Size:** %d bytes (%.2f KB)\n", stat.Size(), float64(stat.Size())/1024)
	output += fmt.Sprintf("**Modified:** %s\n", stat.ModTime().Format("2006-01-02 15:04:05"))
	output += fmt.Sprintf("**Extension:** %s\n", filepath.Ext(path))

	// Detect format from extension
	ext := strings.ToLower(filepath.Ext(path))
	formats := map[string]string{
		".jpg":  "JPEG",
		".jpeg": "JPEG",
		".png":  "PNG",
		".gif":  "GIF",
		".webp": "WebP",
		".bmp":  "BMP",
		".svg":  "SVG",
		".ico":  "ICO",
	}

	if format, ok := formats[ext]; ok {
		output += fmt.Sprintf("**Format:** %s\n", format)
	}

	output += "\n**Note:** For detailed image analysis (dimensions, EXIF, etc.), "
	output += "use image processing libraries or tools like ImageMagick.\n"

	return &ToolResult{
		Output:  output,
		IsError: false,
	}, nil
}

// resolvePath resolves a path relative to workDir
func resolvePath(workDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(workDir, path)
}
