import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(values):
    emit("start", i="-", values=show(values))
    for i in range(len(values) - 1, 0, -1):
        values[i] = values[i - 1]
        emit("copy", i=i, values=show(values))
    values[0] = 0
    emit("fill", i=0, values=show(values))
    return values


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n = tokens[0]
    result(show(solve(tokens[1:1 + n])))


main()
