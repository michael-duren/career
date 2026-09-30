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

    static String showLong(long[] list) {
        StringJoiner out = new StringJoiner(",", "[", "]");
        for (long value : list) out.add(Long.toString(value));
        return out.toString();
    }

    static long[] solve(int[] values, int[][] queries) {
        long[] prefix = new long[values.length + 1];
        for (int k = 0; k < values.length; k++) prefix[k + 1] = prefix[k] + values[k];
        emit("prefix", "l", "-", "r", "-", "answer", "-", "prefix", showLong(prefix));
        long[] answers = new long[queries.length];
        for (int k = 0; k < queries.length; k++) {
            int l = queries[k][0], r = queries[k][1];
            answers[k] = prefix[r + 1] - prefix[l];
            emit("query", "l", str(l), "r", str(r), "answer", Long.toString(answers[k]), "prefix", showLong(prefix));
        }
        return answers;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int[] values = new int[in.nextInt()];
        for (int k = 0; k < values.length; k++) values[k] = in.nextInt();
        int[][] queries = new int[in.nextInt()][2];
        for (int[] query : queries) { query[0] = in.nextInt(); query[1] = in.nextInt(); }
        result(showLong(solve(values, queries)));
    }
}
