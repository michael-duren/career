import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def show_grid(grid):
    return "[" + ",".join(show(row) for row in grid) + "]"


def solve(grid):
    rows, cols = len(grid), len(grid[0])
    marks = []
    for r in range(rows):
        for c in range(cols):
            if grid[r][c] == 1:
                marks.append((r, c))
                emit("mark", r=r, c=c, marks=len(marks), grid=show_grid(grid))

    for r, c in marks:
        if r + 1 < rows:
            grid[r + 1][c] = 1
            emit("light", r=r + 1, c=c, marks=len(marks), grid=show_grid(grid))

    emit("done", r="-", c="-", marks=len(marks), grid=show_grid(grid))
    return grid


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    rows, cols = tokens[0], tokens[1]
    grid = [tokens[2 + r * cols:2 + (r + 1) * cols] for r in range(rows)]
    result(show_grid(solve(grid)))


main()
