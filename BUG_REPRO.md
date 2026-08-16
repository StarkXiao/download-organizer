# Bug reproduction

创建大写扩展名的常见文件。

```bash
mkdir -p fixture-004
printf sample > fixture-004/PHOTO.JPG
printf sample > fixture-004/REPORT.PDF
go run . --dir fixture-004 --dry-run
```

实际结果：大写扩展名可能被归入 Other。
预期结果：JPG 应归入 Images，PDF 应归入 Documents。
