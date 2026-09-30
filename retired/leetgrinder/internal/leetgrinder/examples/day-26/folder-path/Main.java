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
        List<String> stack = new ArrayList<>();
        emit("start", "token", "-", "stack", showStack(stack));
        for (String token : tokens) {
            if (token.equals("..")) {
                if (!stack.isEmpty()) stack.remove(stack.size() - 1);
                emit("up", "token", token, "stack", showStack(stack));
            } else if (token.equals(".")) {
                emit("stay", "token", token, "stack", showStack(stack));
            } else {
                stack.add(token);
                emit("enter", "token", token, "stack", showStack(stack));
            }
        }

        emit("done", "token", "-", "stack", showStack(stack));
        return "/" + String.join("/", stack);
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        List<String> tokens = new ArrayList<>();
        while (in.hasNext()) tokens.add(in.next());
        result(solve(tokens));
    }
}
