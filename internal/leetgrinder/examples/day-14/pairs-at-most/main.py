import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(values, limit):
    count = 0
    emit("start", i="-", j="-", count=count)
    for i in range(len(values)):
        lo, hi = i + 1, len(values)
        while lo < hi:
            mid = lo + (hi - lo) // 2
            if values[i] + values[mid] <= limit:
                lo = mid + 1
            else:
                hi = mid
        count += lo - i - 1
        emit("row", i=i, j=lo, count=count)

    emit("done", i="-", j="-", count=count)
    return count


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n, target = tokens[0], tokens[1]
    values = tokens[2:2 + n]
    result(str(solve(values, target)))


main()
