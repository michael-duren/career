import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(values, queries):
    prefix = [0]
    for value in values:
        prefix.append(prefix[-1] + value)
    emit("prefix", l="-", r="-", answer="-", prefix=show(prefix))
    answers = []
    for l, r in queries:
        answers.append(prefix[r + 1] - prefix[l])
        emit("query", l=l, r=r, answer=answers[-1], prefix=show(prefix))
    return answers


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n = tokens[0]
    values = tokens[1:1 + n]
    q = tokens[1 + n]
    queries = [(tokens[2 + n + 2 * k], tokens[3 + n + 2 * k]) for k in range(q)]
    result(show(solve(values, queries)))


main()
