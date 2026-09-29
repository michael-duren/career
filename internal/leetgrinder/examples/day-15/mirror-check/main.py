import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(values, target):
    lo, hi = 0, len(values) - 1
    emit("start", lo=lo, hi=hi)
    while lo < hi:
        if values[lo] != values[hi]:
            emit("mismatch", lo=lo, hi=hi)
            return "false"
        lo += 1
        hi -= 1
        emit("match", lo=lo, hi=hi)

    emit("done", lo=lo, hi=hi)
    return "true"


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n, target = tokens[0], tokens[1]
    values = tokens[2:2 + n]
    result(solve(values, target))


main()
