import json, sys

def emit(event, i, left, right, pairs, valid):
    fields = [("i", i), ("left", left), ("right", right), ("pairs", pairs), ("valid", str(valid).lower())]
    print(json.dumps({"event": event, "variables": [{"name": k, "value": str(v)} for k, v in fields]}, separators=(",", ":")))

def result(value):
    print(json.dumps({"result": str(value).lower()}, separators=(",", ":")))

def solve(a, b):
    checks = 0
    emit("start", -1, "-", "-", checks, True)
    if len(a) != len(b):
        emit("done", len(a), "-", "-", checks, False)
        result(False)
        return
    for i in range(len(a)):
        for j in range(i):
            checks += 1
            if (a[i] == a[j]) != (b[i] == b[j]):
                emit("reject", i, f"i={i},j={j} {a[i]}/{a[j]}", f"i={i},j={j} {b[i]}/{b[j]}", checks, False)
                emit("done", len(a), "-", "-", checks, False)
                result(False)
                return
            emit("compare", i, f"i={i},j={j} {a[i]}/{a[j]}", f"i={i},j={j} {b[i]}/{b[j]}", checks, True)
    emit("done", len(a), "-", "-", checks, True)
    result(True)

if __name__ == "__main__":
    raw = sys.stdin.read().splitlines()
    left = raw[0]
    right = raw[1] if len(raw) > 1 else ""
    solve(list(left), right.split())
