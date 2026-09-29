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
        int left = 0;
        int total = 0;
        int best = 0;
        emit("start", "left", str(left), "right", "-", "total", str(total), "best", str(best));
        for (int right = 0; right < values.length; right++) {
            total += values[right];
            while (total > target) {
                total -= values[left];
                left++;
            }
            best = Math.max(best, right - left + 1);
            emit("step", "left", str(left), "right", str(right), "total", str(total), "best", str(best));
        }

        emit("done", "left", str(left), "right", "-", "total", str(total), "best", str(best));
        return best;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int n = in.nextInt(), target = in.nextInt();
        int[] values = new int[n];
        for (int k = 0; k < n; k++) values[k] = in.nextInt();
        result(String.valueOf(solve(values, target)));
    }
}
