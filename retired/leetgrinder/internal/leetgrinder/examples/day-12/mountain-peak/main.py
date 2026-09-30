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
    emit("start", lo=lo, hi=hi, mid="-")
    while lo < hi:
        mid = lo + (hi - lo) // 2
        if values[mid] < values[mid + 1]:
            lo = mid + 1
            emit("uphill", lo=lo, hi=hi, mid=mid)
        else:
            hi = mid
            emit("downhill", lo=lo, hi=hi, mid=mid)

    emit("peak", lo=lo, hi=hi, mid=lo)
    return lo


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n, target = tokens[0], tokens[1]
    values = tokens[2:2 + n]
    result(str(solve(values, target)))


main()
