import json
import sys

def emit(event, **values):
    print(json.dumps({"event": event, "variables": [{"name": k, "value": str(v)} for k, v in values.items()]}))

def main():
    a = sys.stdin.readline().rstrip("\n")
    b = sys.stdin.readline().rstrip("\n")
    emit("start", a=a, b=b)
    dp = [[0] * (len(b) + 1) for _ in range(len(a) + 1)]
    best, end, best_j = 0, 0, 0
    for i in range(1, len(a) + 1):
        for j in range(1, len(b) + 1):
            if a[i - 1] == b[j - 1]:
                dp[i][j] = dp[i - 1][j - 1] + 1
                if dp[i][j] > best:
                    best, end, best_j = dp[i][j], i, j
            else:
                dp[i][j] = 0
        emit("row", i=i, row=",".join(map(str, dp[i])), best=best, end=end, bestJ=best_j)
    answer = a[end - best:end]
    emit("done", answer=answer)
    print(json.dumps({"result": answer}))

if __name__ == "__main__":
    main()
