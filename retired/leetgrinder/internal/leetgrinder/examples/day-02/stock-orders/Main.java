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

    static String showLetters(int[] counts) {
        StringJoiner out = new StringJoiner(",", "{", "}");
        for (int k = 0; k < 26; k++) if (counts[k] != 0) out.add((char) ('a' + k) + ":" + counts[k]);
        return out.toString();
    }

    static String solve(String stock, String orders) {
        int[] counts = new int[26];
        for (char item : stock.toCharArray()) counts[item - 'a']++;
        emit("stock", "i", "-", "item", "-", "left", "-", "counts", showLetters(counts));
        for (int i = 0; i < orders.length(); i++) {
            char item = orders.charAt(i);
            int left = --counts[item - 'a'];
            emit("order", "i", str(i), "item", String.valueOf(item), "left", str(left), "counts", showLetters(counts));
            if (left < 0) return "short at " + i;
        }

        emit("done", "i", "-", "item", "-", "left", "-", "counts", showLetters(counts));
        return "filled";
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        String stock = in.hasNextLine() ? in.nextLine() : "";
        String orders = in.hasNextLine() ? in.nextLine() : "";
        result(solve(stock.trim(), orders.trim()));
    }
}
