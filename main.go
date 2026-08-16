package main

import (
	"crypto/sha256"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Item struct {
	Path, Name, Category, Target, Reason, Hash string
	Size                                       int64
	Modified                                   time.Time
	DuplicateOf                                string
	Cleanup                                    bool
}

var datePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(20\d{2})[-_年]?(1[0-2]|0?[1-9])[-_月]?([12]\d|3[01]|0?[1-9])`),
	regexp.MustCompile(`(?i)(20\d{2})(0[1-9]|1[0-2])([0-2]\d|3[01])`),
}

var groups = map[string]map[string]bool{
	"Documents":  {".doc": true, ".docx": true, ".pdf": true, ".txt": true, ".md": true, ".rtf": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true, ".csv": true},
	"Images":     {".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".svg": true, ".heic": true, ".bmp": true},
	"Videos":     {".mp4": true, ".mkv": true, ".mov": true, ".avi": true, ".webm": true, ".flv": true},
	"Audio":      {".mp3": true, ".wav": true, ".flac": true, ".m4a": true, ".aac": true, ".ogg": true},
	"Archives":   {".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".xz": true, ".7z": true, ".rar": true},
	"Installers": {".dmg": true, ".pkg": true, ".exe": true, ".msi": true, ".deb": true, ".rpm": true, ".apk": true},
}

func main() {
	dir := flag.String("dir", "", "要整理的下载目录（默认使用当前用户的 Downloads）")
	csvPath := flag.String("csv", "", "将建议导出到 CSV 文件")
	dryRun := flag.Bool("dry-run", true, "只输出建议，不执行移动或清理（默认开启）")
	apply := flag.Bool("apply", false, "执行建议的移动和安全清理")
	cleanupAge := flag.Duration("cleanup-age", 7*24*time.Hour, "临时文件超过此时长才列为可清理项")
	flag.Parse()
	if *cleanupAge < 0 {
		fatal(fmt.Errorf("--cleanup-age 不能为负数"))
	}
	if *apply {
		*dryRun = false
	}
	root := *dir
	if root == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			fatal(err)
		}
		root = filepath.Join(h, "Downloads")
	}
	root, _ = filepath.Abs(root)
	items, err := scan(root, *cleanupAge)
	if err != nil {
		fatal(err)
	}
	findDuplicates(items)
	printReport(items, *dryRun)
	if *csvPath != "" {
		if err := writeCSV(*csvPath, items); err != nil {
			fatal(err)
		}
		fmt.Printf("\nCSV 已写入 %s\n", *csvPath)
	}
	if !*dryRun {
		if err := applyChanges(root, items); err != nil {
			fatal(err)
		}
	}
}

func scan(root string, cleanupAge time.Duration) ([]*Item, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("读取目录 %s: %w", root, err)
	}
	var items []*Item
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		name := e.Name()
		path := filepath.Join(root, name)
		cat := category(name)
		date, dateReason := dateFor(name, info.ModTime())
		target := filepath.Join(root, cat, date.Format("2006-01"), name)
		cleanup := isTemp(name) && time.Since(info.ModTime()) >= cleanupAge
		reason := fmt.Sprintf("扩展名归类为 %s；%s", cat, dateReason)
		if cleanup {
			reason += "；临时文件且已超过清理期限"
		}
		items = append(items, &Item{Path: path, Name: name, Category: cat, Target: target, Reason: reason, Size: info.Size(), Modified: info.ModTime(), Cleanup: cleanup})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func category(name string) string {
	if isTemp(name) {
		return "Temporary"
	}
	ext := strings.ToLower(filepath.Ext(name))
	for cat, exts := range groups {
		if exts[ext] {
			return cat
		}
	}
	return "Other"
}

func isTemp(name string) bool {
	l := strings.ToLower(name)
	for _, suffix := range []string{".crdownload", ".part", ".tmp", ".temp", ".download", ".swp", ".ds_store"} {
		if strings.HasSuffix(l, suffix) {
			return true
		}
	}
	return strings.HasPrefix(name, "~$") || strings.HasSuffix(name, "~")
}

func dateFor(name string, mod time.Time) (time.Time, string) {
	for _, p := range datePatterns {
		m := p.FindStringSubmatch(name)
		if len(m) == 4 {
			y, _ := atoi(m[1])
			mo, _ := atoi(m[2])
			d, _ := atoi(m[3])
			if t, err := time.Parse("2006-1-2", fmt.Sprintf("%d-%d-%d", y, mo, d)); err == nil {
				return t, "文件名包含日期"
			}
		}
	}
	return mod, "文件名无日期，使用修改时间"
}

func atoi(s string) (int, error) { var n int; _, err := fmt.Sscanf(s, "%d", &n); return n, err }

func findDuplicates(items []*Item) {
	byHash := map[string]*Item{}
	for _, it := range items {
		f, err := os.Open(it.Path)
		if err != nil {
			continue
		}
		h := sha256.New()
		_, err = io.CopyN(h, f, 4096)
		if err == io.EOF {
			err = nil
		}
		f.Close()
		if err != nil {
			continue
		}
		it.Hash = fmt.Sprintf("%x", h.Sum(nil))
		if first, ok := byHash[it.Hash]; ok {
			it.DuplicateOf = first.Path
		} else {
			byHash[it.Hash] = it
		}
	}
}

func printReport(items []*Item, dry bool) {
	mode := "dry-run（不会改动文件）"
	if !dry {
		mode = "apply（将执行移动和清理）"
	}
	fmt.Printf("下载目录整理建议 [%s]\n\n", mode)
	for _, it := range items {
		action := fmt.Sprintf("移动到 %s", it.Target)
		if it.DuplicateOf != "" {
			action = "可能重复，参考 " + it.DuplicateOf
		}
		if it.Cleanup {
			action = "可安全清理"
		}
		fmt.Printf("- %-32s | %-12s | %s\n  %s\n", it.Name, it.Category, action, it.Reason)
	}
	if len(items) == 0 {
		fmt.Println("目录中没有可处理的文件。")
	}
}

func writeCSV(path string, items []*Item) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err = w.Write([]string{"name", "path", "category", "target", "reason", "size", "modified", "duplicate_of", "cleanup"}); err != nil {
		return err
	}
	for _, it := range items {
		if err = w.Write([]string{it.Name, it.Path, it.Category, it.Target, it.Reason, fmt.Sprint(it.Size), it.Modified.Format(time.RFC3339), it.DuplicateOf, fmt.Sprint(it.Cleanup)}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func applyChanges(root string, items []*Item) error {
	for _, it := range items {
		if it.Cleanup {
			if err := os.Remove(it.Path); err != nil {
				return fmt.Errorf("清理 %s: %w", it.Path, err)
			}
			continue
		}
		if filepath.Clean(it.Path) == filepath.Clean(it.Target) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(it.Target), 0755); err != nil {
			return err
		}
		if _, err := os.Stat(it.Target); err == nil {
			continue
		}
		if err := os.Rename(it.Path, it.Target); err != nil {
			return fmt.Errorf("移动 %s: %w", it.Name, err)
		}
	}
	fmt.Printf("\n已应用整理建议（根目录：%s）\n", root)
	return nil
}

func fatal(err error) { fmt.Fprintln(os.Stderr, "错误:", err); os.Exit(1) }
