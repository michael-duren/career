import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(a, b):
    merged = []
    i, j = 0, 0
    emit("start", i=i, j=j, merged=show(merged))
    while i < len(a) or j < len(b):
        if j == len(b) or (i < len(a) and a[i] <= b[j]):
            merged.append(a[i])
            i += 1
            emit("take-a", i=i, j=j, merged=show(merged))
        else:
            merged.append(b[j])
            j += 1
            emit("take-b", i=i, j=j, merged=show(merged))

    emit("done", i=i, j=j, merged=show(merged))
    return merged


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n = tokens[0]
    a = tokens[1:1 + n]
    m = tokens[1 + n]
    b = tokens[2 + n:2 + n + m]
    result(show(solve(a, b)))


main()
