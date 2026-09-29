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

    static long solve(int[] values, long limit) {
        long count = 0;
        emit("start", "i", "-", "j", "-", "count", String.valueOf(count));
        for (int i = 0; i < values.length; i++) {
            int lo = i + 1, hi = values.length;
            while (lo < hi) {
                int mid = lo + (hi - lo) / 2;
                if (values[i] + values[mid] <= limit) lo = mid + 1;
                else hi = mid;
            }
            count += lo - i - 1;
            emit("row", "i", str(i), "j", str(lo), "count", String.valueOf(count));
        }

        emit("done", "i", "-", "j", "-", "count", String.valueOf(count));
        return count;
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
