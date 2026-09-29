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
        if token == "..":
            if stack:
                stack.pop()
            emit("up", token=token, stack=show_stack(stack))
        elif token == ".":
            emit("stay", token=token, stack=show_stack(stack))
        else:
            stack.append(token)
            emit("enter", token=token, stack=show_stack(stack))

    emit("done", token="-", stack=show_stack(stack))
    return "/" + "/".join(stack)


def main():
    result(solve(sys.stdin.read().split()))


main()
