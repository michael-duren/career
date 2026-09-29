import java.util.*;

public class Main {
  static String q(String s) {
    return "\"" + s.replace("\\", "\\\\").replace("\"", "\\\"") + "\"";
  }
  static void emit(String event, String... fields) {
    StringBuilder out =
        new StringBuilder("{\"event\":" + q(event) + ",\"variables\":[");
    for (int i = 0; i < fields.length; i += 2) {
      if (i > 0)
        out.append(',');
      out.append("{\"name\":")
          .append(q(fields[i]))
          .append(",\"value\":")
          .append(q(fields[i + 1]))
          .append('}');
    }
    System.out.println(out.append("]}"));
  }
  static void result(String value) {
    System.out.println("{\"result\":" + q(value) + "}");
  }
  static String s(int n) { return Integer.toString(n); }
  static String s(char c) { return Character.toString(c); }

  static String edgeState(TreeMap<Integer, Integer> edges) {
    StringJoiner out = new StringJoiner(",", "{", "}");
    for (var entry : edges.entrySet())
      out.add(entry.getKey() + ":" + entry.getValue());
    return out.toString();
  }
  static void solve(List<int[]> rows) {
    int width = 0;
    for (int b : rows.get(0))
      width += b;
    int answer;
    TreeMap<Integer, Integer> edges = new TreeMap<>();
    int best = 0;
    emit("start", "x", "0", "row", "-1", "state", "{}", "best", "0", "crossed",
         s(rows.size()));
    for (int r = 0; r < rows.size(); r++) {
      int x = 0, j = 0;
      int[] bricks = rows.get(r);
      for (; j + 1 < bricks.length; j++) {
        x += bricks[j];
        edges.put(x, edges.getOrDefault(x, 0) + 1);
        best = Math.max(best, edges.get(x));
        emit("edge", "x", s(x), "row", s(r), "state", edgeState(edges), "best",
             s(best), "crossed", s(rows.size() - best));
      }
    }
    emit("done", "x", s(width), "row", "-1", "state", edgeState(edges), "best",
         s(best), "crossed", s(rows.size() - best));
    answer = rows.size() - best;
    result(s(answer));
  }
  public static void main(String[] args) {
    Scanner in = new Scanner(System.in);
    int n = in.nextInt();
    List<int[]> rows = new ArrayList<>();
    for (int r = 0; r < n; r++) {
      int k = in.nextInt();
      int[] row = new int[k];
      for (int j = 0; j < k; j++)
        row[j] = in.nextInt();
      rows.add(row);
    }
    solve(rows);
  }
}
