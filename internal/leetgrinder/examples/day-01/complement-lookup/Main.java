import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Scanner;
import java.util.TreeMap;

public class Main {
    static String listText(List<Integer> values) {
        StringBuilder text = new StringBuilder("[");
        for (int value : values) {
            if (text.length() > 1) text.append(',');
            text.append(value);
        }
        return text.append(']').toString();
    }

    static String shown(Map<Integer, Integer> seen) {
        StringBuilder text = new StringBuilder("{");
        for (var entry : seen.entrySet()) {
            if (text.length() > 1) text.append(',');
            text.append(entry.getKey()).append(':').append(entry.getValue());
        }
        return text.append('}').toString();
    }

    static void emit(String event, String... fields) {
        StringBuilder text = new StringBuilder("{\"event\":\"").append(event).append("\",\"variables\":[");
        for (int j = 0; j < fields.length; j += 2) {
            if (j > 0) text.append(',');
            text.append("{\"name\":\"").append(fields[j]).append("\",\"value\":\"").append(fields[j + 1]).append("\"}");
        }
        System.out.println(text.append("]}"));
    }

    public static void main(String[] args) {
        Scanner input = new Scanner(System.in);
        int count = input.nextInt();
        int target = input.nextInt();
        List<Integer> values = new ArrayList<>();
        for (int j = 0; j < count; j++) values.add(input.nextInt());
        Map<Integer, Integer> seen = new TreeMap<>();
        emit("start", "values", listText(values), "target", Integer.toString(target), "seen", shown(seen));
        List<Integer> answer = new ArrayList<>();
        for (int i = 0; i < values.size(); i++) {
            int value = values.get(i);
            int need = target - value;
            emit("lookup", "i", Integer.toString(i), "value", Integer.toString(value), "need", Integer.toString(need), "seen", shown(seen));
            Integer earlier = seen.get(need);
            if (earlier != null) {
                emit("match", "i", Integer.toString(i), "need", Integer.toString(need), "earlier", Integer.toString(earlier), "seen", shown(seen));
                answer = List.of(earlier, i);
                break;
            }
            seen.put(value, i);
            emit("insert", "i", Integer.toString(i), "seen", shown(seen));
        }
        emit("done", "result", listText(answer));
        System.out.println("{\"result\":\"" + listText(answer) + "\"}");
    }
}
