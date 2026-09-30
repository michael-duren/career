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

    static String solve(int[] values, int target) {
        int lo = 0, hi = values.length - 1;
        emit("start", "lo", str(lo), "hi", str(hi));
        while (lo < hi) {
            if (values[lo] != values[hi]) {
                emit("mismatch", "lo", str(lo), "hi", str(hi));
                return "false";
            }
            lo++;
            hi--;
            emit("match", "lo", str(lo), "hi", str(hi));
        }

        emit("done", "lo", str(lo), "hi", str(hi));
        return "true";
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int n = in.nextInt(), target = in.nextInt();
        int[] values = new int[n];
        for (int k = 0; k < n; k++) values[k] = in.nextInt();
        result(solve(values, target));
    }
}
