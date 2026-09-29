import json
import sys

def emit(event, **values):
    print(json.dumps({"event": event, "variables": [{"name": k, "value": str(v)} for k, v in values.items()]}))

def main():
    a = sys.stdin.readline().rstrip("\n")
    b = sys.stdin.readline().rstrip("\n")
    emit("start", a=a, b=b)
    dp = [[0] * (len(b) + 1) for _ in range(len(a) + 1)]
    for i in range(1, len(a) + 1):
        for j in range(1, len(b) + 1):
            if a[i - 1] == b[j - 1]:
                dp[i][j] = dp[i - 1][j - 1] + 1
            else:
                dp[i][j] = max(dp[i - 1][j], dp[i][j - 1])
        emit("row", i=i, row=",".join(map(str, dp[i])))
    answer = str(dp[len(a)][len(b)])
    emit("done", answer=answer)
    print(json.dumps({"result": answer}))

if __name__ == "__main__":
    main()
