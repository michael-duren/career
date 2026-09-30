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

    static String showSet(Set<Character> letters) {
        StringJoiner out = new StringJoiner(",", "{", "}");
        for (char letter : letters) out.add(String.valueOf(letter));
        return out.toString();
    }

    static String solve(String[] grid) {
        int n = grid.length;
        List<Set<Character>> rows = new ArrayList<>(), cols = new ArrayList<>();
        for (int k = 0; k < n; k++) { rows.add(new TreeSet<>()); cols.add(new TreeSet<>()); }
        emit("start", "r", "-", "c", "-", "letter", "-", "row", "-", "col", "-");
        for (int r = 0; r < n; r++) {
            for (int c = 0; c < n; c++) {
                char letter = grid[r].charAt(c);
                if (letter == '.') continue;
                if (rows.get(r).contains(letter) || cols.get(c).contains(letter)) {
                    emit("clash", "r", str(r), "c", str(c), "letter", String.valueOf(letter), "row", showSet(rows.get(r)), "col", showSet(cols.get(c)));
                    return "false";
                }
                rows.get(r).add(letter);
                cols.get(c).add(letter);
                emit("add", "r", str(r), "c", str(c), "letter", String.valueOf(letter), "row", showSet(rows.get(r)), "col", showSet(cols.get(c)));
            }
        }

        emit("done", "r", "-", "c", "-", "letter", "-", "row", "-", "col", "-");
        return "true";
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        String[] grid = new String[in.nextInt()];
        for (int k = 0; k < grid.length; k++) grid[k] = in.next();
        result(solve(grid));
    }
}
