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

    static String showCounts(Map<String, Integer> counts) {
        StringJoiner out = new StringJoiner(",", "{", "}");
        for (Map.Entry<String, Integer> entry : counts.entrySet()) if (entry.getValue() != 0) out.add(entry.getKey() + ":" + entry.getValue());
        return out.toString();
    }

    static int solve(String[] socks) {
        Map<String, Integer> waiting = new TreeMap<>();
        int pairs = 0;
        emit("start", "sock", "-", "waiting", showCounts(waiting), "pairs", str(pairs));
        for (String sock : socks) {
            waiting.merge(sock, 1, Integer::sum);
            if (waiting.get(sock) == 2) {
                waiting.put(sock, 0);
                pairs++;
                emit("pair", "sock", sock, "waiting", showCounts(waiting), "pairs", str(pairs));
            } else {
                emit("wait", "sock", sock, "waiting", showCounts(waiting), "pairs", str(pairs));
            }
        }

        emit("done", "sock", "-", "waiting", showCounts(waiting), "pairs", str(pairs));
        return pairs;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        String[] socks = new String[in.nextInt()];
        for (int k = 0; k < socks.length; k++) socks[k] = in.next();
        result(str(solve(socks)));
    }
}
