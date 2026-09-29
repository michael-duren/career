import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(nums, target):
    counts = {}
    pairs = 0
    emit("start", i="-", value="-", need="-", matches="-", pairs=pairs, counts=show_counts(counts))
    for i, value in enumerate(nums):
        need = target - value
        matches = counts.get(need, 0)
        pairs += matches
        emit("count", i=i, value=value, need=need, matches=matches, pairs=pairs, counts=show_counts(counts))
        counts[value] = counts.get(value, 0) + 1
        emit("store", i=i, value=value, need=need, matches=matches, pairs=pairs, counts=show_counts(counts))

    emit("done", i="-", value="-", need="-", matches="-", pairs=pairs, counts=show_counts(counts))
    return pairs


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n, target = tokens[0], tokens[1]
    result(str(solve(tokens[2:2 + n], target)))


def show_counts(counts):
    return "{" + ",".join(f"{key}:{counts[key]}" for key in sorted(counts)) + "}"


main()
