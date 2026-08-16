# Bug reproduction

指定 CSV 输出路径并检查生成文件内容。

```bash
mkdir -p fixture-003
printf sample > fixture-003/report.txt
go run . --dir fixture-003 --dry-run --csv fixture-003/report.csv
wc -c fixture-003/report.csv
```

实际结果：命令成功但 CSV 可能为空或缺少记录。
预期结果：CSV 应包含表头和扫描结果。
