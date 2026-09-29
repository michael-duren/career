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
    k = target
    total = 0
    for right in range(k):
        total += values[right]
    best = total
    emit("first", right=k - 1, total=total, best=best)
    for right in range(k, len(values)):
        total += values[right] - values[right - k]
        best = max(best, total)
        emit("slide", right=right, total=total, best=best)

    emit("done", right="-", total=total, best=best)
    return best


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n, target = tokens[0], tokens[1]
    values = tokens[2:2 + n]
    result(str(solve(values, target)))


main()
