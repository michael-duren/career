import json
import sys


def emit(event, **state):
    print(json.dumps({"event": event, "variables": [
        {"name": name, "value": str(value)} for name, value in state.items()
    ]}, separators=(",", ":")))


def shown(seen):
    return "{" + ",".join(f"{value}:{index}" for value, index in sorted(seen.items())) + "}"


def pair_text(pair):
    return "[" + ",".join(map(str, pair)) + "]"


def main():
    tokens = list(map(int, sys.stdin.read().split()))
    count, target = tokens[:2]
    values = tokens[2:2 + count]
    seen = {}
    emit("start", values=pair_text(values), target=target, seen=shown(seen))
    answer = []
    for i, value in enumerate(values):
        need = target - value
        emit("lookup", i=i, value=value, need=need, seen=shown(seen))
        if need in seen:
            earlier = seen[need]
            emit("match", i=i, need=need, earlier=earlier, seen=shown(seen))
            answer = [earlier, i]
            break
        seen[value] = i
        emit("insert", i=i, seen=shown(seen))
    emit("done", result=pair_text(answer))
    print(json.dumps({"result": pair_text(answer)}, separators=(",", ":")))


if __name__ == "__main__":
    main()
