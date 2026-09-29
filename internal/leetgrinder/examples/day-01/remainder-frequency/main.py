import json
import sys


def emit(event, **state):
    variables = [{"name": name, "value": str(value)} for name, value in state.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def shown(counts):
    return "{" + ",".join(f"{r}:{count}" for r, count in enumerate(counts) if count) + "}"


def list_text(values):
    return "[" + ",".join(map(str, values)) + "]"


def main():
    tokens = list(map(int, sys.stdin.read().split()))
    size = tokens[0]
    songs = tokens[1:1 + size]
    counts = [0] * 60
    pairs = 0
    emit("start", songs=list_text(songs), counts=shown(counts), pairs=pairs)
    for i, duration in enumerate(songs):
        remainder = duration % 60
        need = (60 - remainder) % 60
        matches = counts[need]
        emit("lookup", i=i, duration=duration, remainder=remainder, need=need,
             matches=matches, pairs=pairs, counts=shown(counts))
        pairs += matches
        emit("count", i=i, need=need, matches=matches, pairs=pairs, counts=shown(counts))
        counts[remainder] += 1
        emit("store", i=i, remainder=remainder, pairs=pairs, counts=shown(counts))
    emit("done", pairs=pairs, counts=shown(counts))
    print(json.dumps({"result": str(pairs)}, separators=(",", ":")))


if __name__ == "__main__":
    main()
