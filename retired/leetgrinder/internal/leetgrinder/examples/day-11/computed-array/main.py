import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(n, target):
    lo, hi = 0, n
    emit("start", lo=lo, hi=hi, mid="-", value="-")
    while lo <= hi:
        mid = lo + (hi - lo) // 2
        value = mid * mid
        if value == target:
            emit("found", lo=lo, hi=hi, mid=mid, value=value)
            return mid
        if value < target:
            lo = mid + 1
            emit("go-right", lo=lo, hi=hi, mid=mid, value=value)
        else:
            hi = mid - 1
            emit("go-left", lo=lo, hi=hi, mid=mid, value=value)

    emit("absent", lo=lo, hi=hi, mid="-", value="-")
    return -1


def main():
    n, target = (int(token) for token in sys.stdin.read().split())
    result(str(solve(n, target)))


main()
