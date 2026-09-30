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

    static String showGrid(int[][] grid) {
        StringBuilder out = new StringBuilder("[");
        for (int r = 0; r < grid.length; r++) out.append(r > 0 ? "," : "").append(show(grid[r]));
        return out.append("]").toString();
    }

    static int[][] solve(int[][] grid) {
        int n = grid.length;
        emit("start", "r", "-", "c", "-", "grid", showGrid(grid));
        for (int r = 0; r < n; r++) {
            for (int c = r + 1; c < n; c++) {
                int moved = grid[r][c]; grid[r][c] = grid[c][r]; grid[c][r] = moved;
                emit("swap", "r", str(r), "c", str(c), "grid", showGrid(grid));
            }
        }

        emit("done", "r", "-", "c", "-", "grid", showGrid(grid));
        return grid;
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int rows = in.nextInt(), cols = in.nextInt();
        int[][] grid = new int[rows][cols];
        for (int r = 0; r < rows; r++)
            for (int c = 0; c < cols; c++) grid[r][c] = in.nextInt();
        result(showGrid(solve(grid)));
    }
}
