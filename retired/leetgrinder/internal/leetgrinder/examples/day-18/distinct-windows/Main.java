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

    static int solve(String text, int k) {
        Map<Character, Integer> counts = new HashMap<>();
        int repeats = 0;
        int good = 0;
        emit("start", "right", "-", "window", "", "repeats", str(repeats), "good", str(good));
        for (int right = 0; right < text.length(); right++) {
            char letter = text.charAt(right);
            if (counts.merge(letter, 1, Integer::sum) == 2) repeats++;
            if (right >= k) {
                char old = text.charAt(right - k);
                if (counts.merge(old, -1, Integer::sum) == 1) repeats--;
            }
            if (right >= k - 1) {
                if (repeats == 0) good++;
                emit("window", "right", str(right), "window", text.substring(right - k + 1, right + 1), "repeats", str(repeats), "good", str(good));
            }
        }

        emit("done", "right", "-", "window", "", "repeats", str(repeats), "good", str(good));
        return good;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        String text = in.next();
        int k = in.nextInt();
        result(str(solve(text, k)));
    }
}
