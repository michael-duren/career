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

    static int[] solve(int[] first, int[] second) {
        Set<Integer> stops = new HashSet<>();
        int position = 0;
        for (int k = 0; k < first.length; k++) {
            position += first[k];
            stops.add(position);
            emit("stop", "route", "A", "length", str(first[k]), "position", str(position), "shared", "[]");
        }
        List<Integer> shared = new ArrayList<>();
        position = 0;
        for (int k = 0; k < second.length; k++) {
            position += second[k];
            if (stops.contains(position)) shared.add(position);
            emit("check", "route", "B", "length", str(second[k]), "position", str(position), "shared", show(toArray(shared)));
        }

        emit("done", "route", "-", "length", "-", "position", "-", "shared", show(toArray(shared)));
        return toArray(shared);
    }

    static int[] toArray(List<Integer> values) {
        return values.stream().mapToInt(Integer::intValue).toArray();
    }

    static int[] parse(String line) {
        return Arrays.stream(line.trim().split("\\s+")).mapToInt(Integer::parseInt).toArray();
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int[] first = parse(in.nextLine());
        int[] second = parse(in.nextLine());
        result(show(solve(first, second)));
    }
}
