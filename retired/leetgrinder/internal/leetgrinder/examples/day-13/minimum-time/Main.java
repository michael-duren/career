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

    static long tripsBy(long time, int[] rates) {
        long trips = 0;
        for (int rate : rates) trips += time / rate;
        return trips;
    }

    static long solve(int[] rates, long needed) {
        long lo = 1, hi = (long) Arrays.stream(rates).min().getAsInt() * needed;
        emit("start", "lo", String.valueOf(lo), "hi", String.valueOf(hi), "mid", "-", "trips", "-");
        while (lo < hi) {
            long mid = lo + (hi - lo) / 2;
            long trips = tripsBy(mid, rates);
            if (trips >= needed) {
                hi = mid;
                emit("enough", "lo", String.valueOf(lo), "hi", String.valueOf(hi), "mid", String.valueOf(mid), "trips", String.valueOf(trips));
            } else {
                lo = mid + 1;
                emit("short", "lo", String.valueOf(lo), "hi", String.valueOf(hi), "mid", String.valueOf(mid), "trips", String.valueOf(trips));
            }
        }

        emit("done", "lo", String.valueOf(lo), "hi", String.valueOf(hi), "mid", "-", "trips", String.valueOf(tripsBy(lo, rates)));
        return lo;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int n = in.nextInt();
        long needed = in.nextLong();
        int[] rates = new int[n];
        for (int k = 0; k < n; k++) rates[k] = in.nextInt();
        result(String.valueOf(solve(rates, needed)));
    }
}
