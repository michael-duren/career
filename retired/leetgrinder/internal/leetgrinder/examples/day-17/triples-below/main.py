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
    count = 0
    emit("start", i="-", lo="-", hi="-", count=count)
    for i in range(len(values) - 2):
        lo, hi = i + 1, len(values) - 1
        emit("anchor", i=i, lo=lo, hi=hi, count=count)
        while lo < hi:
            if values[i] + values[lo] + values[hi] < target:
                count += hi - lo
                lo += 1
                emit("count", i=i, lo=lo, hi=hi, count=count)
            else:
                hi -= 1
                emit("too-big", i=i, lo=lo, hi=hi, count=count)

    emit("done", i="-", lo="-", hi="-", count=count)
    return count


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n, target = tokens[0], tokens[1]
    values = tokens[2:2 + n]
    result(str(solve(values, target)))


main()
