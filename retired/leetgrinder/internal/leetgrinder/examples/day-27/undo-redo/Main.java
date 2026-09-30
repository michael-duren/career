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

    static String showStack(List<String> stack) {
        return "[" + String.join(",", stack) + "]";
    }

    static String solve(List<String> tokens) {
        List<String> text = new ArrayList<>(), redo = new ArrayList<>();
        emit("start", "token", "-", "text", showStack(text), "redo", showStack(redo));
        for (String token : tokens) {
            if (token.equals("undo")) {
                if (!text.isEmpty()) redo.add(text.remove(text.size() - 1));
                emit("undo", "token", token, "text", showStack(text), "redo", showStack(redo));
            } else if (token.equals("redo")) {
                if (!redo.isEmpty()) text.add(redo.remove(redo.size() - 1));
                emit("redo", "token", token, "text", showStack(text), "redo", showStack(redo));
            } else {
                text.add(token);
                redo.clear();
                emit("type", "token", token, "text", showStack(text), "redo", showStack(redo));
            }
        }

        emit("done", "token", "-", "text", showStack(text), "redo", showStack(redo));
        return text.isEmpty() ? "(empty)" : String.join("", text);
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        List<String> tokens = new ArrayList<>();
        while (in.hasNext()) tokens.add(in.next());
        result(solve(tokens));
    }
}
