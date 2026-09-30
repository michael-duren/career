import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(grid, limit):
    r, c = len(grid) - 1, 0
    count = 0
    emit("start", r=r, c=c, count=count)
    while r >= 0 and c < len(grid[0]):
        if grid[r][c] <= limit:
            count += r + 1
            c += 1
            emit("take-column", r=r, c=c, count=count)
        else:
            r -= 1
            emit("go-up", r=r, c=c, count=count)

    emit("done", r=r, c=c, count=count)
    return count


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    rows, cols, limit = tokens[0], tokens[1], tokens[2]
    grid = [tokens[3 + k * cols:3 + (k + 1) * cols] for k in range(rows)]
    result(str(solve(grid, limit)))


main()
