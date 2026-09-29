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
    order = []
    emit("start", side="-", order=show(order))
    for c in range(cols):
        order.append(grid[0][c])
    emit("top", side="top", order=show(order))
    for r in range(1, rows):
        order.append(grid[r][cols - 1])
    emit("right", side="right", order=show(order))
    if rows > 1:
        for c in range(cols - 2, -1, -1):
            order.append(grid[rows - 1][c])
        emit("bottom", side="bottom", order=show(order))
    if cols > 1:
        for r in range(rows - 2, 0, -1):
            order.append(grid[r][0])
        emit("left", side="left", order=show(order))

    emit("done", side="-", order=show(order))
    return order


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    rows, cols = tokens[0], tokens[1]
    grid = [tokens[2 + r * cols:2 + (r + 1) * cols] for r in range(rows)]
    result(show(solve(grid)))


main()
