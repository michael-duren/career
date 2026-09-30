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

    static int[] solve(int[] values) {
        emit("start", "i", "-", "key", "-", "j", "-", "values", show(values));
        for (int i = 1; i < values.length; i++) {
            int key = values[i];
            int j = i - 1;
            emit("take", "i", str(i), "key", str(key), "j", str(j), "values", show(values));
            while (j >= 0 && values[j] > key) {
                values[j + 1] = values[j];
                emit("shift", "i", str(i), "key", str(key), "j", str(j), "values", show(values));
                j--;
            }
            values[j + 1] = key;
            emit("place", "i", str(i), "key", str(key), "j", str(j), "values", show(values));
        }

        emit("done", "i", str(values.length), "key", "-", "j", "-", "values", show(values));
        return values;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int n = in.nextInt();
        int[] values = new int[n];
        for (int k = 0; k < n; k++) values[k] = in.nextInt();
        result(show(solve(values)));
    }
}
