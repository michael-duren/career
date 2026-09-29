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

    static int solve(int[] values, int target) {
        int count = 0;
        emit("start", "i", "-", "lo", "-", "hi", "-", "count", str(count));
        for (int i = 0; i + 2 < values.length; i++) {
            int lo = i + 1, hi = values.length - 1;
            emit("anchor", "i", str(i), "lo", str(lo), "hi", str(hi), "count", str(count));
            while (lo < hi) {
                if (values[i] + values[lo] + values[hi] < target) {
                    count += hi - lo;
                    lo++;
                    emit("count", "i", str(i), "lo", str(lo), "hi", str(hi), "count", str(count));
                } else {
                    hi--;
                    emit("too-big", "i", str(i), "lo", str(lo), "hi", str(hi), "count", str(count));
                }
            }
        }

        emit("done", "i", "-", "lo", "-", "hi", "-", "count", str(count));
        return count;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int n = in.nextInt(), target = in.nextInt();
        int[] values = new int[n];
        for (int k = 0; k < n; k++) values[k] = in.nextInt();
        result(String.valueOf(solve(values, target)));
    }
}
