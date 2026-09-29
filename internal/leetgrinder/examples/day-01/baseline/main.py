import json
import sys


def emit(event, **state):
    print(json.dumps({"event": event, "variables": [
        {"name": name, "value": str(value)} for name, value in state.items()
    ]}, separators=(",", ":")))


def list_text(values):
    return "[" + ",".join(map(str, values)) + "]"


def main():
    tokens = list(map(int, sys.stdin.read().split()))
    count, target = tokens[:2]
    values = tokens[2:2 + count]
    emit("start", values=list_text(values), target=target)
    answer = []
    for i in range(count):
        emit("outer", i=i, first=values[i])
        for j in range(i + 1, count):
            total = values[i] + values[j]
            emit("compare", i=i, j=j, left=values[i], right=values[j], sum=total)
            if total == target:
                emit("match", i=i, j=j)
                answer = [i, j]
                break
        if answer:
            break
    emit("done", result=list_text(answer))
    print(json.dumps({"result": list_text(answer)}, separators=(",", ":")))


if __name__ == "__main__":
    main()
