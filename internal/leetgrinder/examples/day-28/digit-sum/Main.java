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

    static int digitSum(int n, int depth) {
        emit("call", "n", str(n), "depth", str(depth), "result", "-");
        if (n < 10) {
            emit("base", "n", str(n), "depth", str(depth), "result", str(n));
            return n;
        }
        int total = digitSum(n / 10, depth + 1) + n % 10;
        emit("return", "n", str(n), "depth", str(depth), "result", str(total));
        return total;
    }

    static int solve(int n) { return digitSum(n, 0); }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        result(str(solve(in.nextInt())));
    }
}
