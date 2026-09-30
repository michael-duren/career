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

    static class Node {
        int value;
        Node next;
        Node(int value, Node next) { this.value = value; this.next = next; }
    }

    static Node build(int[] values) {
        Node head = null;
        for (int k = values.length - 1; k >= 0; k--) head = new Node(values[k], head);
        return head;
    }

    static String render(Node node) {
        StringJoiner out = new StringJoiner("->");
        for (; node != null; node = node.next) out.add(String.valueOf(node.value));
        return out.length() == 0 ? "empty" : out.toString();
    }

    static String solve(int[] values, int target) {
        Node head = build(values);
        Node slow = head, fast = head.next;
        while (fast.next != null) {
            slow = slow.next;
            fast = fast.next.next;
        }
        Node second = slow.next;
        slow.next = null;
        Node prev = null;
        while (second != null) {
            Node following = second.next;
            second.next = prev;
            prev = second;
            second = following;
        }
        emit("halves", "first", render(head), "second", render(prev), "best", "-");
        int best = 0;
        for (Node a = head, b = prev; a != null; a = a.next, b = b.next) {
            best = Math.max(best, a.value + b.value);
            emit("pair", "first", str(a.value), "second", str(b.value), "best", str(best));
        }

        emit("done", "first", "-", "second", "-", "best", str(best));
        return str(best);
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        int n = in.nextInt(), target = in.nextInt();
        int[] values = new int[n];
        for (int k = 0; k < n; k++) values[k] = in.nextInt();
        result(solve(values, target));
    }
}
