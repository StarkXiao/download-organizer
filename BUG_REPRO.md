# Bug reproduction

创建一个文件名不含日期、但修改时间明显属于过去月份的文件。

```bash
mkdir -p fixture-002
printf sample > fixture-002/report.txt
touch -t 202401150000 fixture-002/report.txt
go run . --dir fixture-002 --dry-run
```

实际结果：建议目标月份可能是当前月份。
预期结果：建议目标月份应来自文件修改时间。
