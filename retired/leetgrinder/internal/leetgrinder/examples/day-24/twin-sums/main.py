import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


class Node:
    def __init__(self, value, next=None):
        self.value = value
        self.next = next


def build(values):
    head = None
    for value in reversed(values):
        head = Node(value, head)
    return head


def render(node):
    parts = []
    while node:
        parts.append(str(node.value))
        node = node.next
    return "->".join(parts) or "empty"

def solve(values, target):
    head = build(values)
    slow, fast = head, head.next
    while fast.next:
        slow = slow.next
        fast = fast.next.next
    second, slow.next = slow.next, None
    prev = None
    while second:
        second.next, prev, second = prev, second, second.next
    emit("halves", first=render(head), second=render(prev), best="-")
    best = 0
    a, b = head, prev
    while a:
        best = max(best, a.value + b.value)
        emit("pair", first=str(a.value), second=str(b.value), best=best)
        a, b = a.next, b.next

    emit("done", first="-", second="-", best=best)
    return str(best)


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n, target = tokens[0], tokens[1]
    values = tokens[2:2 + n]
    result(solve(values, target))


main()
