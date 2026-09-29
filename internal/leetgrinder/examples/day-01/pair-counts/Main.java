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

    static String showCounts(Map<Integer, Integer> counts) {
        return counts.toString().replace("=", ":").replace(", ", ",");
    }

    static long solve(int[] nums, int target) {
        Map<Integer, Integer> counts = new TreeMap<>();
        long pairs = 0;
        emit("start", "i", "-", "value", "-", "need", "-", "matches", "-", "pairs", Long.toString(pairs), "counts", showCounts(counts));
        for (int i = 0; i < nums.length; i++) {
            int value = nums[i];
            int need = target - value;
            int matches = counts.getOrDefault(need, 0);
            pairs += matches;
            emit("count", "i", str(i), "value", str(value), "need", str(need), "matches", str(matches), "pairs", Long.toString(pairs), "counts", showCounts(counts));
            counts.merge(value, 1, Integer::sum);
            emit("store", "i", str(i), "value", str(value), "need", str(need), "matches", str(matches), "pairs", Long.toString(pairs), "counts", showCounts(counts));
        }

        emit("done", "i", "-", "value", "-", "need", "-", "matches", "-", "pairs", Long.toString(pairs), "counts", showCounts(counts));
        return pairs;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int n = in.nextInt(), target = in.nextInt();
        int[] nums = new int[n];
        for (int k = 0; k < n; k++) nums[k] = in.nextInt();
        result(Long.toString(solve(nums, target)));
    }
}
