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

    static long solve(long n) {
        long lo = 1, hi = n;
        emit("start", "lo", String.valueOf(lo), "hi", String.valueOf(hi), "mid", "-", "blocks", "-", "enough", "-");
        while (lo < hi) {
            long mid = lo + (hi - lo) / 2;
            long blocks = mid * (mid + 1) / 2;
            if (blocks >= n) {
                hi = mid;
                emit("yes", "lo", String.valueOf(lo), "hi", String.valueOf(hi), "mid", String.valueOf(mid), "blocks", String.valueOf(blocks), "enough", "true");
            } else {
                lo = mid + 1;
                emit("no", "lo", String.valueOf(lo), "hi", String.valueOf(hi), "mid", String.valueOf(mid), "blocks", String.valueOf(blocks), "enough", "false");
            }
        }

        emit("done", "lo", String.valueOf(lo), "hi", String.valueOf(hi), "mid", "-", "blocks", String.valueOf(lo * (lo + 1) / 2), "enough", "true");
        return lo;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        result(String.valueOf(solve(in.nextLong())));
    }
}
