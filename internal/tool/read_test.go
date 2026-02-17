package tool

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsBinaryFile_KnownExtensions(t *testing.T) {
	binaryExts := []string{".zip", ".exe", ".dll", ".so", ".class", ".pyc", ".wasm", ".db"}
	for _, ext := range binaryExts {
		if !isBinaryFile(ext, nil) {
			t.Errorf("expected %s to be detected as binary by extension", ext)
		}
	}
}

func TestIsBinaryFile_TextExtensions(t *testing.T) {
	textContent := []byte("Hello, this is a normal text file.\nWith multiple lines.\n")
	textExts := []string{".go", ".py", ".js", ".txt", ".md", ".html", ".css"}
	for _, ext := range textExts {
		if isBinaryFile(ext, textContent) {
			t.Errorf("expected %s with text content to NOT be detected as binary", ext)
		}
	}
}

func TestIsBinaryFile_NullByte(t *testing.T) {
	data := []byte("hello\x00world")
	if !isBinaryFile(".unknown", data) {
		t.Error("expected file with null byte to be detected as binary")
	}
}

func TestIsBinaryFile_HighNonPrintable(t *testing.T) {
	// Create data that is >30% non-printable
	data := make([]byte, 100)
	for i := range data {
		if i < 40 {
			data[i] = 1 // non-printable control character
		} else {
			data[i] = 'a'
		}
	}
	if !isBinaryFile(".unknown", data) {
		t.Error("expected file with >30% non-printable bytes to be binary")
	}
}

func TestIsBinaryFile_EmptyFile(t *testing.T) {
	if isBinaryFile(".unknown", []byte{}) {
		t.Error("expected empty file to NOT be detected as binary")
	}
}

func TestGetMIMEType(t *testing.T) {
	tests := map[string]string{
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".pdf":  "application/pdf",
		".svg":  "image/svg+xml",
		".go":   "application/octet-stream", // unknown extension
		".wasm": "application/wasm",
	}
	for ext, expected := range tests {
		got := getMIMEType(ext)
		if got != expected {
			t.Errorf("getMIMEType(%q) = %q, want %q", ext, got, expected)
		}
	}
}

func TestIsImageMIME(t *testing.T) {
	if !isImageMIME("image/png") {
		t.Error("image/png should be detected as image")
	}
	if isImageMIME("application/pdf") {
		t.Error("application/pdf should not be detected as image")
	}
}

func TestReadTool_TextFile(t *testing.T) {
	// Create a temporary text file
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tool := ReadTool()
	tc := &ToolContext{WorkDir: dir}
	result, err := tool.Execute(nil, tc, map[string]interface{}{
		"path": path,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}
	if len(result.Attachments) > 0 {
		t.Error("text file should not have attachments")
	}
}

func TestReadTool_BinaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.exe")
	if err := os.WriteFile(path, []byte{0x4D, 0x5A, 0x00, 0x00}, 0644); err != nil {
		t.Fatal(err)
	}

	tool := ReadTool()
	tc := &ToolContext{WorkDir: dir}
	result, err := tool.Execute(nil, tc, map[string]interface{}{
		"path": path,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for binary file")
	}
}

func TestReadTool_ImageFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")
	// Minimal PNG header (just enough to be a file)
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if err := os.WriteFile(path, pngData, 0644); err != nil {
		t.Fatal(err)
	}

	tool := ReadTool()
	tc := &ToolContext{WorkDir: dir, SessionID: "test-session"}
	result, err := tool.Execute(nil, tc, map[string]interface{}{
		"path": path,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error for image: %s", result.Output)
	}
	if result.Output != "Image read successfully" {
		t.Errorf("expected 'Image read successfully', got %q", result.Output)
	}
	if len(result.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(result.Attachments))
	}
	att := result.Attachments[0]
	if att.MIME != "image/png" {
		t.Errorf("expected mime image/png, got %s", att.MIME)
	}
	if att.Type != "file" {
		t.Errorf("expected type 'file', got %s", att.Type)
	}
	if att.SessionID != "test-session" {
		t.Errorf("expected session ID 'test-session', got %s", att.SessionID)
	}
}

func TestReadTool_PDFFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")
	pdfData := []byte("%PDF-1.4 test content")
	if err := os.WriteFile(path, pdfData, 0644); err != nil {
		t.Fatal(err)
	}

	tool := ReadTool()
	tc := &ToolContext{WorkDir: dir}
	result, err := tool.Execute(nil, tc, map[string]interface{}{
		"path": path,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error for PDF: %s", result.Output)
	}
	if result.Output != "PDF read successfully" {
		t.Errorf("expected 'PDF read successfully', got %q", result.Output)
	}
	if len(result.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(result.Attachments))
	}
	if result.Attachments[0].MIME != "application/pdf" {
		t.Errorf("expected mime application/pdf, got %s", result.Attachments[0].MIME)
	}
}

func TestReadTool_SVGFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.svg")
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg"><rect width="100" height="100"/></svg>`
	if err := os.WriteFile(path, []byte(svgContent), 0644); err != nil {
		t.Fatal(err)
	}

	tool := ReadTool()
	tc := &ToolContext{WorkDir: dir}
	result, err := tool.Execute(nil, tc, map[string]interface{}{
		"path": path,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error for SVG: %s", result.Output)
	}
	// SVG should be read as text, not as an image attachment
	if len(result.Attachments) > 0 {
		t.Error("SVG should be read as text, not as attachment")
	}
}

func TestReadTool_FileNotFound(t *testing.T) {
	tool := ReadTool()
	tc := &ToolContext{WorkDir: "/tmp"}
	result, err := tool.Execute(nil, tc, map[string]interface{}{
		"path": "/tmp/nonexistent_file_12345.txt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for non-existent file")
	}
}

func TestReadTool_OffsetAndLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	lines := "line1\nline2\nline3\nline4\nline5"
	if err := os.WriteFile(path, []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	tool := ReadTool()
	tc := &ToolContext{WorkDir: dir}
	result, err := tool.Execute(nil, tc, map[string]interface{}{
		"path":   path,
		"offset": float64(2),
		"limit":  float64(2),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Output)
	}
	// Should contain lines 2-3 only
	if !contains(result.Output, "line2") || !contains(result.Output, "line3") {
		t.Errorf("expected output to contain line2 and line3, got: %s", result.Output)
	}
	if contains(result.Output, "   1 | line1") {
		t.Error("output should not contain line 1")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
