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

    static String showSeen(Map<Integer, Integer> firstSeen) {
        return firstSeen.toString().replace("=", ":").replace(", ", ",");
    }

    static String solve(int[] values, int target) {
        Map<Integer, Integer> firstSeen = new TreeMap<>();
        firstSeen.put(0, -1);
        int hour = 0;
        emit("start", "step", "-", "hour", str(hour), "seen", showSeen(firstSeen));
        for (int step = 0; step < values.length; step++) {
            hour = ((hour + values[step]) % target + target) % target;
            if (firstSeen.containsKey(hour)) {
                emit("repeat", "step", str(step), "hour", str(hour), "seen", showSeen(firstSeen));
                return (firstSeen.get(hour) + 1) + ".." + step;
            }
            firstSeen.put(hour, step);
            emit("move", "step", str(step), "hour", str(hour), "seen", showSeen(firstSeen));
        }

        emit("done", "step", "-", "hour", str(hour), "seen", showSeen(firstSeen));
        return "none";
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int n = in.nextInt(), target = in.nextInt();
        int[] values = new int[n];
        for (int k = 0; k < n; k++) values[k] = in.nextInt();
        result(solve(values, target));
    }
}
