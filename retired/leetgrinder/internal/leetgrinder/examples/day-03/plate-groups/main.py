import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(plates):
    groups = {}
    emit("start", plate="-", key="-", groups=show_groups(groups))
    for plate in plates:
        key = plate.lower().replace("-", "")
        groups.setdefault(key, []).append(plate)
        emit("add", plate=plate, key=key, groups=show_groups(groups))

    emit("done", plate="-", key="-", groups=show_groups(groups))
    return show_groups(groups)


def main():
    tokens = sys.stdin.read().split()
    result(solve(tokens[1:1 + int(tokens[0])]))


def show_groups(groups):
    return "|".join(key + ":" + ",".join(members) for key, members in groups.items())


main()
