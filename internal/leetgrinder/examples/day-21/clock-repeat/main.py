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
    first_seen = {0: -1}
    hour = 0
    emit("start", step="-", hour=hour, seen=show_seen(first_seen))
    for step, shift in enumerate(values):
        hour = ((hour + shift) % target + target) % target
        if hour in first_seen:
            emit("repeat", step=step, hour=hour, seen=show_seen(first_seen))
            return f"{first_seen[hour] + 1}..{step}"
        first_seen[hour] = step
        emit("move", step=step, hour=hour, seen=show_seen(first_seen))

    emit("done", step="-", hour=hour, seen=show_seen(first_seen))
    return "none"


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n, target = tokens[0], tokens[1]
    result(solve(tokens[2:2 + n], target))


def show_seen(first_seen):
    return "{" + ",".join(f"{hour}:{first_seen[hour]}" for hour in sorted(first_seen)) + "}"


main()
