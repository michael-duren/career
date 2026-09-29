import json, sys

def emit(event, i, word, key, groups):
    fields = [("i", i), ("word", word), ("key", key), ("groups", groups)]
    print(json.dumps({"event": event, "variables": [{"name": k, "value": str(v)} for k,v in fields]}, separators=(",", ":")))

def result(value):
    print(json.dumps({"result": value}, separators=(",", ":")))

def render(groups):
    return "|".join(",".join(group) for group in groups)

def solve(words):
    keys, groups = {}, []
    emit("start", -1, "-", "-", "-")
    for i, word in enumerate(words):
        counts = [0] * 26
        for letter in word:
            counts[ord(letter)-97] += 1
        key = ",".join(f"{chr(97+i)}:{n}" for i,n in enumerate(counts) if n)
        if key not in keys:
            keys[key] = len(groups)
            groups.append([])
        groups[keys[key]].append(word)
        emit("group", i, word, key, render(groups))
    answer = render(groups)
    emit("done", len(words), "-", "-", answer)
    result(answer)

if __name__ == "__main__":
    tokens = sys.stdin.read().split()
    solve(tokens[1:1+int(tokens[0])])
