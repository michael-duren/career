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
    count = 0
    emit("start", lo=lo, hi=hi, count=count)
    while lo < hi:
        if values[lo] + values[hi] <= target:
            count += hi - lo
            lo += 1
            emit("count", lo=lo, hi=hi, count=count)
        else:
            hi -= 1
            emit("too-big", lo=lo, hi=hi, count=count)

    emit("done", lo=lo, hi=hi, count=count)
    return count


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n, target = tokens[0], tokens[1]
    values = tokens[2:2 + n]
    result(str(solve(values, target)))


main()
