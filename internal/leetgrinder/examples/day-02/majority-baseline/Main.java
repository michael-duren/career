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

  static void solve(int[] values) {
    String answer = "none";
    int lastSeen = 0, lastCandidate = 0;
    emit("start", "i", "-1", "candidate", "-", "tally", "0", "answer", "none");
    for (int i = 0; i < values.length; i++) {
      int candidate = values[i], seen = 0;
      for (int value : values)
        if (value == candidate)
          seen++;
      lastSeen = seen;
      lastCandidate = candidate;
      emit("test", "i", s(i), "candidate", s(candidate), "tally", s(seen),
           "answer", "none");
      if (seen > values.length / 2) {
        answer = s(candidate);
        break;
      }
    }
    emit("done", "i", s(values.length), "candidate", s(lastCandidate), "tally",
         s(lastSeen), "answer", answer);
    result(answer);
  }
  public static void main(String[] args) {
    Scanner in = new Scanner(System.in);
    int n = in.nextInt();
    int[] values = new int[n];
    for (int i = 0; i < n; i++)
      values[i] = in.nextInt();
    solve(values);
  }
}
