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
    best = len(rows)
    emit("start", "x", 0, "row", -1, "state", "outside", "best", best, "crossed", 0)
    for x in range(1, width):
        crossed = 0
        emit("probe", "x", x, "row", -1, "state", "testing", "best", best, "crossed", crossed)
        for r, bricks in enumerate(rows):
            position = 0
            hit = False
            for brick in bricks[:-1]:
                position += brick
                if position == x:
                    hit = True
                    break
            if not hit:
                crossed += 1
            emit("row", "x", x, "row", r, "state", "edge" if hit else "brick", "best", best, "crossed", crossed)
        best = min(best, crossed)
        emit("commit", "x", x, "row", -1, "state", "completed", "best", best, "crossed", crossed)
    emit("done", "x", width, "row", -1, "state", "outside", "best", best, "crossed", best)
    answer = best
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
