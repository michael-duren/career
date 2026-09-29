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

    static int step(int x) { return (x * x + 1) % 10; }

    static String showSet(Set<Integer> seen) { return show(seen.stream().mapToInt(Integer::intValue).toArray()); }

    static int solve(int start) {
        Set<Integer> seen = new TreeSet<>();
        int x = start;
        emit("start", "x", str(x), "seen", showSet(seen));
        while (!seen.contains(x)) {
            seen.add(x);
            x = step(x);
            emit("step", "x", str(x), "seen", showSet(seen));
        }

        emit("repeat", "x", str(x), "seen", showSet(seen));
        return x;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        result(str(solve(in.nextInt())));
    }
}
