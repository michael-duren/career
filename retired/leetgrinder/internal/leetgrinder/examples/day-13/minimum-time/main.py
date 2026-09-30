import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def trips_by(time, rates):
    return sum(time // rate for rate in rates)


def solve(rates, needed):
    lo, hi = 1, min(rates) * needed
    emit("start", lo=lo, hi=hi, mid="-", trips="-")
    while lo < hi:
        mid = lo + (hi - lo) // 2
        trips = trips_by(mid, rates)
        if trips >= needed:
            hi = mid
            emit("enough", lo=lo, hi=hi, mid=mid, trips=trips)
        else:
            lo = mid + 1
            emit("short", lo=lo, hi=hi, mid=mid, trips=trips)

    emit("done", lo=lo, hi=hi, mid="-", trips=trips_by(lo, rates))
    return lo


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n, needed = tokens[0], tokens[1]
    result(str(solve(tokens[2:2 + n], needed)))


main()
