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
    emit("start", i="-", key="-", j="-", values=show(values))
    for i in range(1, len(values)):
        key = values[i]
        j = i - 1
        emit("take", i=i, key=key, j=j, values=show(values))
        while j >= 0 and values[j] > key:
            values[j + 1] = values[j]
            emit("shift", i=i, key=key, j=j, values=show(values))
            j -= 1
        values[j + 1] = key
        emit("place", i=i, key=key, j=j, values=show(values))

    emit("done", i=len(values), key="-", j="-", values=show(values))
    return values


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n = tokens[0]
    result(show(solve(tokens[1:1 + n])))


main()
