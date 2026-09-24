#!/usr/bin/env python3
# jdiff.py BEFORE AFTER: every JSON path whose value differs between two
# dumps, with both values; a list holding the same items in another order is
# marked REORDERED rather than walked.
import json
import sys


def walk(a, b, path, out):
    if isinstance(a, dict) and isinstance(b, dict):
        for k in sorted(set(a) | set(b)):
            walk(a.get(k, "<absent>"), b.get(k, "<absent>"), f"{path}.{k}", out)
    elif isinstance(a, list) and isinstance(b, list) and len(a) == len(b) \
            and sorted(map(json.dumps, a)) != sorted(map(json.dumps, b)):
        for i, (x, y) in enumerate(zip(a, b)):
            walk(x, y, f"{path}[{i}]", out)
    elif a != b:
        same = isinstance(a, list) and isinstance(b, list) \
            and sorted(map(json.dumps, a)) == sorted(map(json.dumps, b))
        out.append(f"{path}: {'REORDERED ' if same else ''}{json.dumps(a)[:90]} -> {json.dumps(b)[:90]}")


out = []
walk(json.load(open(sys.argv[1])), json.load(open(sys.argv[2])), "", out)
print("\n".join(out))
