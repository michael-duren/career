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
    int best = rows.size();
    emit("start", "x", "0", "row", "-1", "state", "outside", "best", s(best),
         "crossed", "0");
    for (int x = 1; x < width; x++) {
      int crossed = 0;
      emit("probe", "x", s(x), "row", "-1", "state", "testing", "best", s(best),
           "crossed", s(crossed));
      for (int r = 0; r < rows.size(); r++) {
        int position = 0;
        boolean hit = false;
        int[] bricks = rows.get(r);
        for (int j = 0; j + 1 < bricks.length; j++) {
          position += bricks[j];
          if (position == x) {
            hit = true;
            break;
          }
        }
        if (!hit)
          crossed++;
        emit("row", "x", s(x), "row", s(r), "state", hit ? "edge" : "brick",
             "best", s(best), "crossed", s(crossed));
      }
      best = Math.min(best, crossed);
      emit("commit", "x", s(x), "row", "-1", "state", "completed", "best",
           s(best), "crossed", s(crossed));
    }
    emit("done", "x", s(width), "row", "-1", "state", "outside", "best",
         s(best), "crossed", s(best));
    answer = best;
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
