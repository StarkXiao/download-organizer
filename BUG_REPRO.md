# Bug reproduction

准备两个文件：它们前 4096 字节完全相同，但第 4097 字节之后不同。

```bash
python3 - <<'PY'
from pathlib import Path
p = Path('fixture-001')
p.mkdir(exist_ok=True)
(p / 'first.bin').write_bytes(b'A' * 4096 + b'first')
(p / 'second.bin').write_bytes(b'A' * 4096 + b'second')
PY
go run . --dir fixture-001 --dry-run
```

实际结果：两个文件可能被报告为重复。
预期结果：只有完整内容相同的文件才应被报告为重复。
