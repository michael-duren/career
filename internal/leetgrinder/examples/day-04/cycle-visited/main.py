import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(start):
    seen = set()
    x = start
    emit("start", x=x, seen=show(sorted(seen)))
    while x not in seen:
        seen.add(x)
        x = step(x)
        emit("step", x=x, seen=show(sorted(seen)))

    emit("repeat", x=x, seen=show(sorted(seen)))
    return x


def step(x):
    return (x * x + 1) % 10


def main():
    result(str(solve(int(sys.stdin.read()))))


main()
