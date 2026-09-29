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

    static String solve(String word) {
        int[] counts = new int[26];
        emit("start", "i", "-", "letter", "-", "count", "-", "repeated", "");
        for (int i = 0; i < word.length(); i++) {
            char letter = word.charAt(i);
            counts[letter - 'a']++;
            emit("count", "i", str(i), "letter", String.valueOf(letter), "count", str(counts[letter - 'a']), "repeated", "");
        }
        StringBuilder repeated = new StringBuilder();
        for (int k = 0; k < 26; k++) {
            if (counts[k] > 1) {
                repeated.append((char) ('a' + k));
                emit("repeat", "i", "-", "letter", String.valueOf((char) ('a' + k)), "count", str(counts[k]), "repeated", repeated.toString());
            }
        }

        emit("done", "i", "-", "letter", "-", "count", "-", "repeated", repeated.toString());
        return repeated.length() == 0 ? "none" : repeated.toString();
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        result(solve(in.hasNext() ? in.next() : ""));
    }
}
