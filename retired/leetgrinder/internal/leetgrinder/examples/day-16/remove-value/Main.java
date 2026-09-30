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

    static int[] solve(int[] values, int target) {
        int write = 0;
        emit("start", "read", "-", "write", str(write), "values", show(values));
        for (int read = 0; read < values.length; read++) {
            if (values[read] != target) {
                values[write] = values[read];
                write++;
                emit("keep", "read", str(read), "write", str(write), "values", show(values));
            } else {
                emit("drop", "read", str(read), "write", str(write), "values", show(values));
            }
        }

        emit("done", "read", "-", "write", str(write), "values", show(values));
        return Arrays.copyOf(values, write);
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int n = in.nextInt(), target = in.nextInt();
        int[] values = new int[n];
        for (int k = 0; k < n; k++) values[k] = in.nextInt();
        result(show(solve(values, target)));
    }
}
