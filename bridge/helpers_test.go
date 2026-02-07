package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"  hello  ", "hello"},
		{"\nhello\n", "hello"},
		{"\r\nhello\r\n", "hello"},
		{"\t hello \t", "hello"},
		{"", ""},
		{"   ", ""},
		{"\n\r\t", ""},
		{"  middle spaces  ", "middle spaces"},
	}
	for _, tt := range tests {
		got := trimSpace(tt.input)
		if got != tt.want {
			t.Errorf("trimSpace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestReadChunksFile(t *testing.T) {
	dir := t.TempDir()

	// Normal file with content
	path := filepath.Join(dir, "test.chunks")
	content := "chunk1\nchunk2\n\nchunk3\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	chunks, err := readChunksFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if chunks[0] != "chunk1" || chunks[1] != "chunk2" || chunks[2] != "chunk3" {
		t.Errorf("unexpected chunks: %v", chunks)
	}

	// Empty file
	emptyPath := filepath.Join(dir, "empty.chunks")
	if err := os.WriteFile(emptyPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	chunks, err = readChunksFile(emptyPath)
	if err != nil {
		t.Fatalf("unexpected error for empty file: %v", err)
	}
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty file, got %d", len(chunks))
	}

	// Missing file
	_, err = readChunksFile(filepath.Join(dir, "nonexistent.chunks"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestReadChunksFile_WhitespaceLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spaces.chunks")
	content := "  chunk1  \n   \nchunk2\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	chunks, err := readChunksFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0] != "chunk1" {
		t.Errorf("expected trimmed chunk1, got %q", chunks[0])
	}
}
