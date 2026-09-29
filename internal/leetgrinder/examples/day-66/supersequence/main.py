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
    i, j, out = len(a), len(b), []
    while i > 0 or j > 0:
        from_i, from_j = i, j
        upper = str(dp[i - 1][j]) if i else "-"
        left = str(dp[i][j - 1]) if j else "-"
        if i == 0:
            choice = "rest-b"
            out.append(b[j - 1])
            j -= 1
        elif j == 0:
            choice = "rest-a"
            out.append(a[i - 1])
            i -= 1
        elif a[i - 1] == b[j - 1]:
            choice = "match"
            out.append(a[i - 1])
            i -= 1
            j -= 1
        elif dp[i - 1][j] >= dp[i][j - 1]:
            choice = "upper"
            out.append(a[i - 1])
            i -= 1
        else:
            choice = "left"
            out.append(b[j - 1])
            j -= 1
        emit("walk", fromI=from_i, fromJ=from_j, upper=upper, left=left, choice=choice, i=i, j=j, out="".join(out))
    answer = "".join(reversed(out))
    emit("done", answer=answer)
    print(json.dumps({"result": answer}))

if __name__ == "__main__":
    main()
