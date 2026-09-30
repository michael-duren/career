import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(n):
    lo, hi = 1, n
    emit("start", lo=lo, hi=hi, mid="-", blocks="-", enough="-")
    while lo < hi:
        mid = lo + (hi - lo) // 2
        blocks = mid * (mid + 1) // 2
        if blocks >= n:
            hi = mid
            emit("yes", lo=lo, hi=hi, mid=mid, blocks=blocks, enough="true")
        else:
            lo = mid + 1
            emit("no", lo=lo, hi=hi, mid=mid, blocks=blocks, enough="false")

    emit("done", lo=lo, hi=hi, mid="-", blocks=lo * (lo + 1) // 2, enough="true")
    return lo


def main():
    result(str(solve(int(sys.stdin.read()))))


main()
