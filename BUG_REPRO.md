# Bug reproduction

创建一个刚生成的临时下载文件，并使用默认清理期限执行 apply。

```bash
mkdir -p fixture-005
printf partial > fixture-005/incomplete.part
go run . --dir fixture-005 --apply --cleanup-age 168h
```

实际结果：刚创建的临时文件也可能被列为清理项并删除。
预期结果：只有超过 cleanup-age 的临时文件才能被清理。
