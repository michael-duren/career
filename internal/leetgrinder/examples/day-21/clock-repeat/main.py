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
    seen = {0}
    hour = 0
    emit("start", step="-", hour=hour, seen=show(sorted(seen)))
    for step, shift in enumerate(values):
        hour = ((hour + shift) % target + target) % target
        if hour in seen:
            emit("repeat", step=step, hour=hour, seen=show(sorted(seen)))
            return "yes"
        seen.add(hour)
        emit("move", step=step, hour=hour, seen=show(sorted(seen)))

    emit("done", step="-", hour=hour, seen=show(sorted(seen)))
    return "no"


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n, target = tokens[0], tokens[1]
    values = tokens[2:2 + n]
    result(solve(values, target))


main()
