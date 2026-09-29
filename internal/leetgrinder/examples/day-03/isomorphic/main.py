import json, sys

def emit(event, i, left, right, pairs, valid):
    fields = [("i", i), ("left", left), ("right", right), ("pairs", pairs), ("valid", str(valid).lower())]
    print(json.dumps({"event": event, "variables": [{"name": k, "value": str(v)} for k, v in fields]}, separators=(",", ":")))

def result(value):
    print(json.dumps({"result": str(value).lower()}, separators=(",", ":")))

def solve(a, b):
    forward, reverse = {}, {}
    emit("start", -1, "-", "-", 0, True)
    if len(a) != len(b):
        emit("done", len(a), "-", "-", 0, False)
        result(False)
        return
    for i, (x, y) in enumerate(zip(a, b)):
        if (x in forward and forward[x] != y) or (y in reverse and reverse[y] != x):
            emit("reject", i, x, y, len(forward), False)
            emit("done", len(a), "-", "-", len(forward), False)
            result(False)
            return
        forward[x], reverse[y] = y, x
        emit("pair", i, x, y, len(forward), True)
    emit("done", len(a), "-", "-", len(forward), True)
    result(True)

if __name__ == "__main__":
    left, right = sys.stdin.read().split()
    solve(*(list(left), list(right)))
