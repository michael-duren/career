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

    static int solve(int[][] grid, int limit) {
        int r = grid.length - 1, c = 0;
        int count = 0;
        emit("start", "r", str(r), "c", str(c), "count", str(count));
        while (r >= 0 && c < grid[0].length) {
            if (grid[r][c] <= limit) {
                count += r + 1;
                c++;
                emit("take-column", "r", str(r), "c", str(c), "count", str(count));
            } else {
                r--;
                emit("go-up", "r", str(r), "c", str(c), "count", str(count));
            }
        }

        emit("done", "r", str(r), "c", str(c), "count", str(count));
        return count;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int rows = in.nextInt(), cols = in.nextInt(), limit = in.nextInt();
        int[][] grid = new int[rows][cols];
        for (int r = 0; r < rows; r++)
            for (int c = 0; c < cols; c++) grid[r][c] = in.nextInt();
        result(str(solve(grid, limit)));
    }
}
