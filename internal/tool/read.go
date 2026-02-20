package tool

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadTool reads file contents with optional offset and limit.
// Handles images and PDFs as base64 attachments, rejects other binary files.
func ReadTool() *ToolDef {
	return &ToolDef{
		Name:        "read",
		Description: "Read file contents with optional offset/limit. Images and PDFs returned as attachments.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The absolute or relative file path to read",
				},
				"offset": map[string]interface{}{
					"type":        "integer",
					"description": "Line number to start reading from (1-based). Default: 1",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of lines to read. Default: 2000",
				},
			},
			"required": []string{"path"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			path, _ := input["path"].(string)
			if path == "" {
				return &ToolResult{Output: "Error: path is required", IsError: true}, nil
			}

			// Resolve relative paths
			if !filepath.IsAbs(path) && tc.WorkDir != "" {
				path = filepath.Join(tc.WorkDir, path)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					return handleFileNotFound(path)
				}
				return &ToolResult{Output: fmt.Sprintf("Error reading file: %v", err), IsError: true}, nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			mime := getMIMEType(ext)

			// Check if this is an image (excluding SVG which is text-based)
			if isImageMIME(mime) && mime != "image/svg+xml" {
				return handleImageFile(path, data, mime, tc)
			}

			// Check if this is a PDF
			if mime == "application/pdf" {
				return handlePDFFile(path, data, mime, tc)
			}

			// Check if this is a binary file
			if isBinaryFile(ext, data) {
				return &ToolResult{
					Output:  fmt.Sprintf("Cannot read binary file: %s", path),
					IsError: true,
				}, nil
			}

			// Text file - proceed with normal reading
			return readTextFile(path, data, input)
		},
	}
}

// handleFileNotFound suggests similar files when a file is not found.
func handleFileNotFound(path string) (*ToolResult, error) {
	dir := filepath.Dir(path)
	entries, _ := os.ReadDir(dir)
	suggestions := []string{}
	base := filepath.Base(path)
	prefix := base
	if len(prefix) > 3 {
		prefix = prefix[:3]
	}
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Name()), strings.ToLower(prefix)) {
			suggestions = append(suggestions, e.Name())
		}
	}
	msg := fmt.Sprintf("File not found: %s", path)
	if len(suggestions) > 0 {
		msg += fmt.Sprintf("\nDid you mean: %s", strings.Join(suggestions, ", "))
	}
	return &ToolResult{Output: msg, IsError: true}, nil
}

// handleImageFile returns an image as a base64 attachment.
func handleImageFile(path string, data []byte, mime string, tc *ToolContext) (*ToolResult, error) {
	b64 := base64.StdEncoding.EncodeToString(data)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mime, b64)

	attachment := FileAttachment{
		ID:       filepath.Base(path),
		Type:     "file",
		MIME:     mime,
		URL:      dataURL,
		Filename: filepath.Base(path),
	}
	if tc != nil {
		attachment.SessionID = tc.SessionID
		attachment.MessageID = tc.MessageID
	}

	return &ToolResult{
		Output:      "Image read successfully",
		Attachments: []FileAttachment{attachment},
	}, nil
}

// handlePDFFile returns a PDF as a base64 attachment.
func handlePDFFile(path string, data []byte, mime string, tc *ToolContext) (*ToolResult, error) {
	b64 := base64.StdEncoding.EncodeToString(data)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mime, b64)

	attachment := FileAttachment{
		ID:       filepath.Base(path),
		Type:     "file",
		MIME:     mime,
		URL:      dataURL,
		Filename: filepath.Base(path),
	}
	if tc != nil {
		attachment.SessionID = tc.SessionID
		attachment.MessageID = tc.MessageID
	}

	return &ToolResult{
		Output:      "PDF read successfully",
		Attachments: []FileAttachment{attachment},
	}, nil
}

// readTextFile reads a text file with offset and limit parameters.
func readTextFile(path string, data []byte, input map[string]interface{}) (*ToolResult, error) {
	content := string(data)
	lines := strings.Split(content, "\n")

	offset := 1
	if v, ok := input["offset"].(float64); ok && v > 0 {
		offset = int(v)
	}
	limit := 2000
	if v, ok := input["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	// Apply offset and limit
	startIdx := offset - 1
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx >= len(lines) {
		return &ToolResult{Output: fmt.Sprintf("Offset %d exceeds file length (%d lines)", offset, len(lines))}, nil
	}
	endIdx := startIdx + limit
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	selectedLines := lines[startIdx:endIdx]
	// Add line numbers
	numbered := make([]string, len(selectedLines))
	for i, line := range selectedLines {
		numbered[i] = fmt.Sprintf("%4d | %s", startIdx+i+1, line)
	}

	result := strings.Join(numbered, "\n")

	// Truncate if too large (20KB limit to save tokens)
	if len(result) > 20*1024 {
		result = result[:20*1024] + "\n... (truncated, file too large)"
	}

	header := fmt.Sprintf("File: %s (%d lines total, showing lines %d-%d)\n\n", path, len(lines), startIdx+1, endIdx)
	return &ToolResult{Output: header + result}, nil
}

// getMIMEType returns the MIME type for a file extension.
func getMIMEType(ext string) string {
	mimeTypes := map[string]string{
		// Images
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".webp": "image/webp",
		".bmp":  "image/bmp",
		".ico":  "image/x-icon",
		".svg":  "image/svg+xml",
		".tiff": "image/tiff",
		".tif":  "image/tiff",
		".avif": "image/avif",
		".heic": "image/heic",
		".heif": "image/heif",

		// PDFs
		".pdf": "application/pdf",

		// Audio
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg",
		".flac": "audio/flac",
		".aac":  "audio/aac",
		".m4a":  "audio/mp4",

		// Video
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".mkv":  "video/x-matroska",

		// Archives
		".zip": "application/zip",
		".tar": "application/x-tar",
		".gz":  "application/gzip",
		".7z":  "application/x-7z-compressed",
		".rar": "application/vnd.rar",

		// Executables
		".exe":   "application/x-msdownload",
		".dll":   "application/x-msdownload",
		".so":    "application/x-sharedlib",
		".dylib": "application/x-sharedlib",
		".wasm":  "application/wasm",

		// Fonts
		".ttf":   "font/ttf",
		".otf":   "font/otf",
		".woff":  "font/woff",
		".woff2": "font/woff2",
		".eot":   "application/vnd.ms-fontobject",

		// Office
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	}

	if m, ok := mimeTypes[ext]; ok {
		return m
	}
	return "application/octet-stream"
}

// isImageMIME checks if a MIME type represents an image.
func isImageMIME(mime string) bool {
	return strings.HasPrefix(mime, "image/")
}

// isBinaryFile detects if a file is binary using a two-tier strategy:
// 1. Extension-based check (fast, deterministic)
// 2. Content-based heuristic (NULL byte or >30% non-printable bytes)
func isBinaryFile(ext string, data []byte) bool {
	// Tier 1: Known binary extensions
	binaryExtensions := map[string]bool{
		// Archives
		".zip": true, ".tar": true, ".gz": true, ".7z": true, ".rar": true,
		".bz2": true, ".xz": true, ".zst": true,
		// Executables and libraries
		".exe": true, ".dll": true, ".so": true, ".o": true, ".a": true,
		".lib": true, ".wasm": true, ".dylib": true,
		// Office formats
		".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
		".ppt": true, ".pptx": true, ".odt": true, ".ods": true, ".odp": true,
		// Java
		".class": true, ".jar": true, ".war": true,
		// Python bytecode
		".pyc": true, ".pyo": true,
		// Other binary
		".bin": true, ".dat": true, ".obj": true,
		// Media (non-image, handled separately)
		".mp3": true, ".mp4": true, ".wav": true, ".avi": true, ".mov": true,
		".mkv": true, ".webm": true, ".flac": true, ".ogg": true, ".aac": true,
		".m4a": true,
		// Fonts
		".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".eot": true,
		// Database
		".db": true, ".sqlite": true, ".sqlite3": true,
		// Disk images
		".iso": true, ".img": true, ".dmg": true,
	}

	if binaryExtensions[ext] {
		return true
	}

	// Tier 2: Content-based heuristic
	if len(data) == 0 {
		return false // Empty files are not binary
	}

	// Check first 4096 bytes
	sampleSize := 4096
	if len(data) < sampleSize {
		sampleSize = len(data)
	}
	sample := data[:sampleSize]

	nonPrintableCount := 0
	for _, b := range sample {
		// NULL byte = definitely binary
		if b == 0 {
			return true
		}
		// Non-printable: below tab (9) or between 14 and 31
		// Printable control chars: tab(9), newline(10), vtab(11), formfeed(12), CR(13)
		if b < 9 || (b > 13 && b < 32) {
			nonPrintableCount++
		}
	}

	// More than 30% non-printable = binary
	return float64(nonPrintableCount)/float64(len(sample)) > 0.3
}
