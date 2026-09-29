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
    int candidate = 0, balance = 0;
    emit("start", "i", "-1", "value", "-", "candidate", "-", "balance", "0",
         "seen", "0");
    for (int i = 0; i < values.length; i++) {
      int value = values[i];
      if (balance == 0) {
        candidate = value;
        balance = 1;
      } else if (candidate == value)
        balance++;
      else
        balance--;
      emit("vote", "i", s(i), "value", s(value), "candidate", s(candidate),
           "balance", s(balance), "seen", "0");
    }
    int seen = 0;
    for (int i = 0; i < values.length; i++) {
      int value = values[i];
      if (value == candidate)
        seen++;
      emit("verify", "i", s(i), "value", s(value), "candidate", s(candidate),
           "balance", s(balance), "seen", s(seen));
    }
    if (seen > values.length / 2)
      answer = s(candidate);
    emit("done", "i", s(values.length), "value", "-", "candidate", s(candidate),
         "balance", s(balance), "seen", s(seen));
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
