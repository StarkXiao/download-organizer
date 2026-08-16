package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDateFor(t *testing.T) {
	got, reason := dateFor("invoice-2024-05-20.pdf", time.Time{})
	if got.Format("2006-01-02") != "2024-05-20" || reason != "文件名包含日期" {
		t.Fatalf("dateFor returned %v, %q", got, reason)
	}
	mod := time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC)
	got, _ = dateFor("invoice.pdf", mod)
	if !got.Equal(mod) {
		t.Fatalf("expected modification time fallback, got %v", got)
	}
}

func TestTempRulesDoNotTreatHiddenConfigAsTemp(t *testing.T) {
	for _, name := range []string{"file.part", "download.crdownload", "~$draft.docx", "notes~"} {
		if !isTemp(name) {
			t.Errorf("expected %q to be temporary", name)
		}
	}
	if isTemp(".env") {
		t.Error(".env must not be considered safe temporary data")
	}
}

func TestScanAndDuplicateDetection(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "report-2024-05-20.txt")
	second := filepath.Join(dir, "copy.txt")
	if err := os.WriteFile(first, []byte("same"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("same"), 0600); err != nil {
		t.Fatal(err)
	}
	items, err := scan(dir, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	findDuplicates(items)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	var duplicate bool
	for _, item := range items {
		duplicate = duplicate || item.DuplicateOf != ""
	}
	if !duplicate {
		t.Fatal("expected duplicate file to be detected")
	}
}
