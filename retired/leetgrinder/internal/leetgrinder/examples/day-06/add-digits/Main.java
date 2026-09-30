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

    static int[] solve(int[] a, int[] b) {
        int i = a.length - 1, j = b.length - 1, carry = 0;
        List<Integer> digits = new ArrayList<>();
        emit("start", "i", str(i), "j", str(j), "carry", str(carry), "digits", show(toArray(digits)));
        while (i >= 0 || j >= 0 || carry > 0) {
            int total = carry;
            if (i >= 0) total += a[i];
            if (j >= 0) total += b[j];
            digits.add(total % 10);
            carry = total / 10;
            emit("column", "i", str(i), "j", str(j), "carry", str(carry), "digits", show(toArray(digits)));
            i--;
            j--;
        }

        Collections.reverse(digits);
        emit("done", "i", str(i), "j", str(j), "carry", str(carry), "digits", show(toArray(digits)));
        return toArray(digits);
    }

    static int[] toArray(List<Integer> values) {
        return values.stream().mapToInt(Integer::intValue).toArray();
    }

    static int[] parse(String line) {
        return Arrays.stream(line.trim().split("\\s+")).mapToInt(Integer::parseInt).toArray();
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int[] a = parse(in.nextLine());
        int[] b = parse(in.nextLine());
        result(show(solve(a, b)));
    }
}
