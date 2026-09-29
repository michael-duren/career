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

  static String countState(int[] counts) {
    StringJoiner out = new StringJoiner(",", "{", "}");
    for (int i = 0; i < 26; i++)
      if (counts[i] != 0)
        out.add((char)('a' + i) + ":" + counts[i]);
    return out.toString();
  }
  static void solve(String note, String magazine) {
    boolean ok = true;
    int[] counts = new int[26];
    emit("start", "i", "-1", "letter", "-", "position", "-1", "state", "{}",
         "ok", "true");
    for (int j = 0; j < magazine.length(); j++) {
      char letter = magazine.charAt(j);
      counts[letter - 'a']++;
      emit("supply", "i", s(j), "letter", s(letter), "position", s(j), "state",
           countState(counts), "ok", "true");
    }
    for (int i = 0; i < note.length(); i++) {
      char letter = note.charAt(i);
      counts[letter - 'a']--;
      ok = counts[letter - 'a'] >= 0;
      emit("request", "i", s(i), "letter", s(letter), "position", "-1", "state",
           countState(counts), "ok", Boolean.toString(ok));
      if (!ok)
        break;
    }
    emit("done", "i", s(note.length()), "letter", "-", "position", "-1",
         "state", countState(counts), "ok", Boolean.toString(ok));
    result(Boolean.toString(ok));
  }
  public static void main(String[] args) {
    Scanner in = new Scanner(System.in);
    solve(in.next(), in.next());
  }
}
