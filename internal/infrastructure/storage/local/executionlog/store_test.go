package executionlog

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReaderWaitsWithoutCreatingFileThenReadsBoundedAppends(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "stages", "one.log")
	reader, err := (Store{}).OpenReader(context.Background(), logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	data, exists, err := reader.Read(context.Background(), 0, 32*1024)
	if err != nil || exists || len(data) != 0 {
		t.Fatalf("missing log read = %q, %v, %v", data, exists, err)
	}
	if _, err := os.Stat(filepath.Dir(logPath)); !os.IsNotExist(err) {
		t.Fatalf("reader created parent directory: %v", err)
	}
	writer, err := (Store{}).Writer(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Close() }()
	content := bytes.Repeat([]byte("a"), 70*1024)
	if _, err := writer.Write(content); err != nil {
		t.Fatal(err)
	}
	var offset int64
	var output []byte
	for {
		data, exists, err := reader.Read(context.Background(), offset, 32*1024)
		if err != nil || !exists || len(data) > 32*1024 {
			t.Fatalf("read = %d, %v, %v", len(data), exists, err)
		}
		if len(data) == 0 {
			break
		}
		output = append(output, data...)
		offset += int64(len(data))
	}
	if !bytes.Equal(output, content) {
		t.Fatal("bounded reads lost content")
	}
	if _, err := writer.Write([]byte("last")); err != nil {
		t.Fatal(err)
	}
	data, _, err = reader.Read(context.Background(), offset, 32*1024)
	if err != nil || string(data) != "last" {
		t.Fatalf("append after EOF = %q, %v", data, err)
	}
}
