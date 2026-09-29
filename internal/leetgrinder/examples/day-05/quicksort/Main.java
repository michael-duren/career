import java.util.*;

public class Main {
    static void emit(String event, String... pairs) {
        StringBuilder out = new StringBuilder("{\"event\":\"" + event + "\",\"variables\":[");
        for (int k = 0; k < pairs.length; k += 2) {
            if (k > 0) out.append(',');
            out.append("{\"name\":\"").append(pairs[k]).append("\",\"value\":\"").append(pairs[k + 1]).append("\"}");
        }
        System.out.println(out.append("]}"));
    }

    static void result(String answer) { System.out.println("{\"result\":\"" + answer + "\"}"); }

    static String str(int value) { return Integer.toString(value); }

    static String show(int[] values) {
        StringBuilder out = new StringBuilder("[");
        for (int k = 0; k < values.length; k++) out.append(k > 0 ? "," : "").append(values[k]);
        return out.append("]").toString();
    }

    static void quicksort(int[] values, int lo, int hi) {
        if (lo >= hi) return;
        int pivot = values[hi];
        int store = lo;
        emit("pivot", "lo", str(lo), "hi", str(hi), "pivot", str(pivot), "store", str(store), "scan", "-", "values", show(values));
        for (int scan = lo; scan < hi; scan++) {
            if (values[scan] < pivot) {
                int moved = values[store]; values[store] = values[scan]; values[scan] = moved;
                store++;
                emit("smaller", "lo", str(lo), "hi", str(hi), "pivot", str(pivot), "store", str(store), "scan", str(scan), "values", show(values));
            } else {
                emit("not-smaller", "lo", str(lo), "hi", str(hi), "pivot", str(pivot), "store", str(store), "scan", str(scan), "values", show(values));
            }
        }
        values[hi] = values[store];
        values[store] = pivot;
        emit("place", "lo", str(lo), "hi", str(hi), "pivot", str(pivot), "store", str(store), "scan", "-", "values", show(values));
        quicksort(values, lo, store - 1);
        quicksort(values, store + 1, hi);
    }

    static int[] solve(int[] values) {
        int last = values.length - 1;
        emit("start", "lo", "0", "hi", str(last), "pivot", "-", "store", "-", "scan", "-", "values", show(values));
        quicksort(values, 0, last);

        emit("done", "lo", "0", "hi", str(last), "pivot", "-", "store", "-", "scan", "-", "values", show(values));
        return values;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int n = in.nextInt();
        int[] values = new int[n];
        for (int k = 0; k < n; k++) values[k] = in.nextInt();
        result(show(solve(values)));
    }
}
