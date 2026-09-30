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

    static long solve(long n, long target) {
        long lo = 0, hi = n;
        emit("start", "lo", String.valueOf(lo), "hi", String.valueOf(hi), "mid", "-", "value", "-");
        while (lo <= hi) {
            long mid = lo + (hi - lo) / 2;
            long value = mid * mid;
            if (value == target) {
                emit("found", "lo", String.valueOf(lo), "hi", String.valueOf(hi), "mid", String.valueOf(mid), "value", String.valueOf(value));
                return mid;
            }
            if (value < target) {
                lo = mid + 1;
                emit("go-right", "lo", String.valueOf(lo), "hi", String.valueOf(hi), "mid", String.valueOf(mid), "value", String.valueOf(value));
            } else {
                hi = mid - 1;
                emit("go-left", "lo", String.valueOf(lo), "hi", String.valueOf(hi), "mid", String.valueOf(mid), "value", String.valueOf(value));
            }
        }

        emit("absent", "lo", String.valueOf(lo), "hi", String.valueOf(hi), "mid", "-", "value", "-");
        return -1;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        long n = in.nextLong(), target = in.nextLong();
        result(String.valueOf(solve(n, target)));
    }
}
