import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(socks):
    waiting = {}
    pairs = 0
    emit("start", sock="-", waiting=show_counts(waiting), pairs=pairs)
    for sock in socks:
        waiting[sock] = waiting.get(sock, 0) + 1
        if waiting[sock] == 2:
            waiting[sock] = 0
            pairs += 1
            emit("pair", sock=sock, waiting=show_counts(waiting), pairs=pairs)
        else:
            emit("wait", sock=sock, waiting=show_counts(waiting), pairs=pairs)

    emit("done", sock="-", waiting=show_counts(waiting), pairs=pairs)
    return pairs


def main():
    tokens = sys.stdin.read().split()
    result(str(solve(tokens[1:1 + int(tokens[0])])))


def show_counts(counts):
    return "{" + ",".join(f"{key}:{counts[key]}" for key in sorted(counts) if counts[key]) + "}"


main()
