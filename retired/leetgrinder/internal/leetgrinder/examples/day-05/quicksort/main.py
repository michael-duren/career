import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def quicksort(values, lo, hi):
    if lo >= hi:
        return
    pivot = values[hi]
    store = lo
    emit("pivot", lo=lo, hi=hi, pivot=pivot, store=store, scan="-", values=show(values))
    for scan in range(lo, hi):
        if values[scan] < pivot:
            values[store], values[scan] = values[scan], values[store]
            store += 1
            emit("smaller", lo=lo, hi=hi, pivot=pivot, store=store, scan=scan, values=show(values))
        else:
            emit("not-smaller", lo=lo, hi=hi, pivot=pivot, store=store, scan=scan, values=show(values))
    values[store], values[hi] = values[hi], values[store]
    emit("place", lo=lo, hi=hi, pivot=pivot, store=store, scan="-", values=show(values))
    quicksort(values, lo, store - 1)
    quicksort(values, store + 1, hi)


def solve(values):
    emit("start", lo=0, hi=len(values) - 1, pivot="-", store="-", scan="-", values=show(values))
    quicksort(values, 0, len(values) - 1)

    emit("done", lo=0, hi=len(values) - 1, pivot="-", store="-", scan="-", values=show(values))
    return values


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n = tokens[0]
    result(show(solve(tokens[1:1 + n])))


main()
