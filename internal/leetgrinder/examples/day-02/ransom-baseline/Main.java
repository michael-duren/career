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

  static String usedState(boolean[] used) {
    List<Integer> positions = new ArrayList<>();
    for (int i = 0; i < used.length; i++)
      if (used[i])
        positions.add(i);
    return positions.toString();
  }
  static void solve(String note, String magazine) {
    boolean ok = true;
    boolean[] used = new boolean[magazine.length()];
    emit("start", "i", "-1", "letter", "-", "position", "-1", "state", "[]",
         "ok", "true");
    for (int i = 0; i < note.length(); i++) {
      boolean found = false;
      for (int j = 0; j < magazine.length(); j++) {
        emit("try", "i", s(i), "letter", s(note.charAt(i)), "position", s(j),
             "state", usedState(used), "ok", "true");
        if (magazine.charAt(j) == note.charAt(i) && !used[j]) {
          used[j] = true;
          found = true;
          emit("take", "i", s(i), "letter", s(note.charAt(i)), "position", s(j),
               "state", usedState(used), "ok", "true");
          break;
        }
      }
      if (!found) {
        ok = false;
        emit("missing", "i", s(i), "letter", s(note.charAt(i)), "position",
             "-1", "state", usedState(used), "ok", "false");
        break;
      }
    }
    emit("done", "i", s(note.length()), "letter", "-", "position", "-1",
         "state", usedState(used), "ok", Boolean.toString(ok));
    result(Boolean.toString(ok));
  }
  public static void main(String[] args) {
    Scanner in = new Scanner(System.in);
    solve(in.next(), in.next());
  }
}
