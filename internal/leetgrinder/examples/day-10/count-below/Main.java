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

    static int solve(int[] values, long target) {
        int lo = 0, hi = values.length;
        emit("start", "lo", str(lo), "hi", str(hi), "mid", "-");
        while (lo < hi) {
            int mid = lo + (hi - lo) / 2;
            if (values[mid] < target) {
                lo = mid + 1;
                emit("below", "lo", str(lo), "hi", str(hi), "mid", str(mid));
            } else {
                hi = mid;
                emit("not-below", "lo", str(lo), "hi", str(hi), "mid", str(mid));
            }
        }

        emit("done", "lo", str(lo), "hi", str(hi), "mid", "-");
        return lo;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int n = in.nextInt();
        long target = in.nextLong();
        int[] values = new int[n];
        for (int k = 0; k < n; k++) values[k] = in.nextInt();
        result(String.valueOf(solve(values, target)));
    }
}
