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

    static String binary(int n, int depth) {
        emit("call", "n", str(n), "depth", str(depth), "result", "-");
        if (n < 2) {
            emit("base", "n", str(n), "depth", str(depth), "result", str(n));
            return str(n);
        }
        String text = binary(n / 2, depth + 1) + (n % 2);
        emit("return", "n", str(n), "depth", str(depth), "result", text);
        return text;
    }

    static String solve(int n) { return binary(n, 0); }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        result(solve(in.nextInt()));
    }
}
