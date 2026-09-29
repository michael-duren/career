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

    static String showGroups(Map<String, List<String>> groups) {
        StringJoiner out = new StringJoiner("|");
        for (Map.Entry<String, List<String>> entry : groups.entrySet()) out.add(entry.getKey() + ":" + String.join(",", entry.getValue()));
        return out.toString();
    }

    static String solve(String[] plates) {
        Map<String, List<String>> groups = new LinkedHashMap<>();
        emit("start", "plate", "-", "key", "-", "groups", showGroups(groups));
        for (String plate : plates) {
            String key = plate.toLowerCase().replace("-", "");
            groups.computeIfAbsent(key, k -> new ArrayList<>()).add(plate);
            emit("add", "plate", plate, "key", key, "groups", showGroups(groups));
        }

        emit("done", "plate", "-", "key", "-", "groups", showGroups(groups));
        return showGroups(groups);
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        String[] plates = new String[in.nextInt()];
        for (int k = 0; k < plates.length; k++) plates[k] = in.next();
        result(solve(plates));
    }
}
