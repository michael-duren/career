import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def show_stack(stack):
    return "[" + ",".join(stack) + "]"

def solve(tokens):
    stack = []
    emit("start", token="-", stack=show_stack(stack))
    for token in tokens:
        if token.startswith("</"):
            name = token[2:-1]
            if not stack or stack[-1] != name:
                emit("mismatch", token=token, stack=show_stack(stack))
                return "false"
            stack.pop()
            emit("close", token=token, stack=show_stack(stack))
        else:
            stack.append(token[1:-1])
            emit("open", token=token, stack=show_stack(stack))

    emit("done", token="-", stack=show_stack(stack))
    return "true" if not stack else "false"


def main():
    result(solve(sys.stdin.read().split()))


main()
