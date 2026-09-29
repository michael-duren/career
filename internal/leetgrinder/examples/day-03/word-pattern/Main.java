import java.util.*;
public class Main {
    static void emit(String event, int i, String left, String right, int pairs, boolean valid) {
        System.out.println("{\"event\":\"" + event + "\",\"variables\":["
            + "{\"name\":\"i\",\"value\":\"" + i + "\"},"
            + "{\"name\":\"left\",\"value\":\"" + left + "\"},"
            + "{\"name\":\"right\",\"value\":\"" + right + "\"},"
            + "{\"name\":\"pairs\",\"value\":\"" + pairs + "\"},"
            + "{\"name\":\"valid\",\"value\":\"" + valid + "\"}]}");
    }
    static void result(boolean value) { System.out.println("{\"result\":\"" + value + "\"}"); }

    public static void main(String[] args) {
        Scanner scanner = new Scanner(System.in); String left = scanner.nextLine(), right = scanner.hasNextLine() ? scanner.nextLine() : "";
        List<String> a = new ArrayList<>(), b = new ArrayList<>(); for (char c : left.toCharArray()) a.add(String.valueOf(c)); Scanner words = new Scanner(right); while (words.hasNext()) b.add(words.next());
        Map<String,String> forward = new HashMap<>(), reverse = new HashMap<>();
        emit("start", -1, "-", "-", 0, true);
        if (a.size() != b.size()) { emit("done", a.size(), "-", "-", 0, false); result(false); return; }
        for (int i=0; i<a.size(); ++i) {
            String x=a.get(i), y=b.get(i);
            if ((forward.containsKey(x) && !forward.get(x).equals(y)) || (reverse.containsKey(y) && !reverse.get(y).equals(x))) {
                emit("reject", i, x, y, forward.size(), false);
                emit("done", a.size(), "-", "-", forward.size(), false);
                result(false); return;
            }
            forward.put(x,y); reverse.put(y,x);
            emit("pair", i, x, y, forward.size(), true);
        }
        emit("done", a.size(), "-", "-", forward.size(), true);
        result(true);
    }
}
