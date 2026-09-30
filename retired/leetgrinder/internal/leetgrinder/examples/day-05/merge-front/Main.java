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

    static int[] solve(int[] a, int[] b) {
        int[] merged = new int[a.length + b.length];
        int i = 0, j = 0, size = 0;
        emit("start", "i", str(i), "j", str(j), "merged", show(Arrays.copyOf(merged, size)));
        while (i < a.length || j < b.length) {
            if (j == b.length || (i < a.length && a[i] <= b[j])) {
                merged[size++] = a[i];
                i++;
                emit("take-a", "i", str(i), "j", str(j), "merged", show(Arrays.copyOf(merged, size)));
            } else {
                merged[size++] = b[j];
                j++;
                emit("take-b", "i", str(i), "j", str(j), "merged", show(Arrays.copyOf(merged, size)));
            }
        }

        emit("done", "i", str(i), "j", str(j), "merged", show(merged));
        return merged;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int[] a = new int[in.nextInt()];
        for (int k = 0; k < a.length; k++) a[k] = in.nextInt();
        int[] b = new int[in.nextInt()];
        for (int k = 0; k < b.length; k++) b[k] = in.nextInt();
        result(show(solve(a, b)));
    }
}
