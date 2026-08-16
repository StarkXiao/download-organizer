# Download Organizer

一个使用 Go 编写的下载目录整理建议工具。它依据扩展名、文件名日期、修改时间和常见临时文件规则生成建议，列出建议移动目标、可能重复文件以及可安全清理项。

## 使用

```bash
go run . --dir ~/Downloads --dry-run
go run . --dir ~/Downloads --csv report.csv
go run . --dir ~/Downloads --apply --cleanup-age 168h
```

默认是 dry-run，不会改变文件。只有显式传入 `--apply` 才会创建 `类别/YYYY-MM` 目录并移动文件，超过 `--cleanup-age` 的临时文件会被删除；同内容文件只标记为可能重复，不会自动删除。
