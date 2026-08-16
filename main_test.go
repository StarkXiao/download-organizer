package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
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

// writeCSV must flush the buffered csv.Writer before the file is closed,
// otherwise the exported file is empty or missing already-written records.
func TestWriteCSV(t *testing.T) {
	dir := t.TempDir()

	// A mix large enough to exceed the csv.Writer buffer (4096 bytes) so a
	// missing flush would truncate the tail, plus enough variety to verify
	// every column is written correctly.
	const n = 200
	items := make([]*Item, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, &Item{
			Path:        filepath.Join(dir, "file-%d.txt"),
			Name:        "file-%d.txt",
			Category:    "Documents",
			Target:      filepath.Join(dir, "Documents", "2024-05", "file-%d.txt"),
			Reason:      "扩展名归类为 Documents；文件名无日期，使用修改时间",
			Size:        int64(i),
			Modified:    time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			DuplicateOf: "",
			Cleanup:     false,
		})
	}
	// Mark one row as a duplicate and another as a cleanup candidate so
	// those columns are non-trivial.
	items[0].DuplicateOf = filepath.Join(dir, "original.txt")
	items[1].Cleanup = true

	csvPath := filepath.Join(dir, "report.csv")
	if err := writeCSV(csvPath, items); err != nil {
		t.Fatalf("writeCSV: %v", err)
	}

	data, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("CSV file is empty; buffered records were not flushed to disk")
	}

	f, err := os.Open(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("read back CSV: %v", err)
	}

	const expectedCols = 9
	wantRows := n + 1 // header + data rows
	if len(records) != wantRows {
		t.Fatalf("expected %d rows (1 header + %d data), got %d", wantRows, n, len(records))
	}

	wantHeader := []string{"name", "path", "category", "target", "reason", "size", "modified", "duplicate_of", "cleanup"}
	for i, c := range records[0] {
		if c != wantHeader[i] {
			t.Errorf("header[%d] = %q, want %q", i, c, wantHeader[i])
		}
	}
	for _, row := range records {
		if len(row) != expectedCols {
			t.Errorf("row %q has %d columns, want %d", row[0], len(row), expectedCols)
		}
	}

	// Spot-check the last data row: if the flush was missing, the tail
	// (including this row) would never reach disk.
	last := records[len(records)-1]
	if last[5] != strconv.Itoa(n-1) {
		t.Errorf("last row size = %q, want %d", last[5], n-1)
	}
	// Spot-check the first row's duplicate_of column carried through.
	if records[1][7] != filepath.Join(dir, "original.txt") {
		t.Errorf("first data row duplicate_of = %q", records[1][7])
	}
	if records[2][8] != "true" {
		t.Errorf("second data row cleanup = %q, want true", records[2][8])
	}
}

// An empty item set must still produce a header-only CSV (and a non-empty
// file), proving the flush runs even when there is nothing to write.
func TestWriteCSVMptyItems(t *testing.T) {
	csvPath := filepath.Join(t.TempDir(), "empty.csv")
	if err := writeCSV(csvPath, nil); err != nil {
		t.Fatalf("writeCSV: %v", err)
	}
	data, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "name,path,category,target,reason,size,modified,duplicate_of,cleanup\n"
	if string(data) != want {
		t.Errorf("empty export = %q, want %q", data, want)
	}
}
