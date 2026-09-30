import json
import sys

def emit(event, **values):
    print(json.dumps({"event": event, "variables": [{"name": k, "value": str(v)} for k, v in values.items()]}))

def main():
    a = sys.stdin.readline().rstrip("\n")
    b = sys.stdin.readline().rstrip("\n")
    emit("start", a=a, b=b)
    def visit(i, j):
        emit("call", i=i, j=j)
        if i == 0 or j == 0:
            return 0
        if a[i - 1] == b[j - 1]:
            return 1 + visit(i - 1, j - 1)
        return max(visit(i - 1, j), visit(i, j - 1))
    answer = str(visit(len(a), len(b)))
    emit("done", answer=answer)
    print(json.dumps({"result": answer}))

if __name__ == "__main__":
    main()
