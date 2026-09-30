import json
import sys
from collections import deque

def compact(value):
    return json.dumps(value, separators=(",", ":"))

def emit(event, depth, current, queue, level, result):
    values = [("depth", str(depth)), ("current", str(current)),
              ("queue", compact(list(queue))), ("level", compact(level)),
              ("result", compact(result))]
    print(compact({"event": event, "variables":
                   [{"name": name, "value": value} for name, value in values]}))

def solve(rows):
    queue = deque([0]) if rows else deque()
    depth = 0
    result = 0  # MODE: depth
    emit("init", depth, -1, queue, [], result)
    while queue and result == 0:
        depth += 1
        count = len(queue)
        level = []
        emit("level_start", depth, -1, queue, level, result)
        for _ in range(count):
            node = queue.popleft()
            value, left, right = rows[node]
            if left == -1 and right == -1:
                result = depth
                emit("leaf", depth, node, queue, level, result)
                break
            if left != -1: queue.append(left)
            if right != -1: queue.append(right)
            emit("visit", depth, node, queue, level, result)
    emit("done", depth, -1, queue, [], result)
    print(compact({"result": compact(result)}))

def main():
    data = list(map(int, sys.stdin.read().split()))
    n = data[0]
    solve([tuple(data[1 + 3*i:4 + 3*i]) for i in range(n)])

if __name__ == "__main__":
    main()
