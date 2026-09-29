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
    left = 0
    total = 0
    best = 0
    emit("start", left=left, right="-", total=total, best=best)
    for right in range(len(values)):
        total += values[right]
        while total > target:
            total -= values[left]
            left += 1
        best = max(best, right - left + 1)
        emit("step", left=left, right=right, total=total, best=best)

    emit("done", left=left, right="-", total=total, best=best)
    return best


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n, target = tokens[0], tokens[1]
    values = tokens[2:2 + n]
    result(str(solve(values, target)))


main()
