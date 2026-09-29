import json
import sys


def emit(event, *fields):
    variables = [{"name": fields[i], "value": str(fields[i+1])} for i in range(0, len(fields), 2)]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))

def result(value):
    print(json.dumps({"result": str(value)}, separators=(",", ":")))

def state_edges(edges):
    return "{" + ",".join(f"{x}:{edges[x]}" for x in sorted(edges)) + "}"

def solve(rows):
    width = sum(rows[0])
    edges = {}
    best = 0
    emit("start", "x", 0, "row", -1, "state", "{}", "best", 0, "crossed", len(rows))
    for r, bricks in enumerate(rows):
        x = 0
        for brick in bricks[:-1]:
            x += brick
            edges[x] = edges.get(x, 0) + 1
            best = max(best, edges[x])
            emit("edge", "x", x, "row", r, "state", state_edges(edges), "best", best, "crossed", len(rows)-best)
    emit("done", "x", width, "row", -1, "state", state_edges(edges), "best", best, "crossed", len(rows)-best)
    answer = len(rows)-best
    result(answer)

if __name__ == "__main__":
    data = list(map(int, sys.stdin.read().split()))
    r = data[0]
    rows = []
    at = 1
    for _ in range(r):
        k = data[at]
        rows.append(data[at+1:at+1+k])
        at += k+1
    solve(rows)
