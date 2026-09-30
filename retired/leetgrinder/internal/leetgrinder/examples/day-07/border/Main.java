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

    static int[] solve(int[][] grid) {
        int rows = grid.length, cols = grid[0].length;
        List<Integer> order = new ArrayList<>();
        emit("start", "side", "-", "order", show(toArray(order)));
        for (int c = 0; c < cols; c++) order.add(grid[0][c]);
        emit("top", "side", "top", "order", show(toArray(order)));
        for (int r = 1; r < rows; r++) order.add(grid[r][cols - 1]);
        emit("right", "side", "right", "order", show(toArray(order)));
        if (rows > 1) {
            for (int c = cols - 2; c >= 0; c--) order.add(grid[rows - 1][c]);
            emit("bottom", "side", "bottom", "order", show(toArray(order)));
        }
        if (cols > 1) {
            for (int r = rows - 2; r > 0; r--) order.add(grid[r][0]);
            emit("left", "side", "left", "order", show(toArray(order)));
        }

        emit("done", "side", "-", "order", show(toArray(order)));
        return toArray(order);
    }

    static int[] toArray(List<Integer> values) {
        return values.stream().mapToInt(Integer::intValue).toArray();
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int rows = in.nextInt(), cols = in.nextInt();
        int[][] grid = new int[rows][cols];
        for (int r = 0; r < rows; r++)
            for (int c = 0; c < cols; c++) grid[r][c] = in.nextInt();
        result(show(solve(grid)));
    }
}
