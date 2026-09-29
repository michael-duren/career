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
    n = len(grid)
    emit("start", r="-", c="-", grid=show_grid(grid))
    for r in range(n):
        for c in range(r + 1, n):
            grid[r][c], grid[c][r] = grid[c][r], grid[r][c]
            emit("swap", r=r, c=c, grid=show_grid(grid))

    emit("done", r="-", c="-", grid=show_grid(grid))
    return grid


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    rows, cols = tokens[0], tokens[1]
    grid = [tokens[2 + r * cols:2 + (r + 1) * cols] for r in range(rows)]
    result(show_grid(solve(grid)))


main()
