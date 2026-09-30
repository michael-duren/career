import java.util.ArrayList;
import java.util.List;
import java.util.Scanner;

public class Main {
    static String listText(List<Integer> values) {
        StringBuilder text = new StringBuilder("[");
        for (int value : values) {
            if (text.length() > 1) text.append(',');
            text.append(value);
        }
        return text.append(']').toString();
    }

    static void emit(String event, String... fields) {
        StringBuilder text = new StringBuilder("{\"event\":\"").append(event).append("\",\"variables\":[");
        for (int k = 0; k < fields.length; k += 2) {
            if (k > 0) text.append(',');
            text.append("{\"name\":\"").append(fields[k]).append("\",\"value\":\"").append(fields[k + 1]).append("\"}");
        }
        System.out.println(text.append("]}"));
    }

    public static void main(String[] args) {
        Scanner input = new Scanner(System.in);
        int count = input.nextInt();
        int target = input.nextInt();
        List<Integer> values = new ArrayList<>();
        for (int k = 0; k < count; k++) values.add(input.nextInt());
        emit("start", "values", listText(values), "target", Integer.toString(target));
        List<Integer> answer = new ArrayList<>();
        for (int i = 0; i < count; i++) {
            emit("outer", "i", Integer.toString(i), "first", Integer.toString(values.get(i)));
            for (int j = i + 1; j < count; j++) {
                long total = (long) values.get(i) + values.get(j);
                emit("compare", "i", Integer.toString(i), "j", Integer.toString(j), "left", Integer.toString(values.get(i)), "right", Integer.toString(values.get(j)), "sum", Long.toString(total));
                if (total == target) {
                    emit("match", "i", Integer.toString(i), "j", Integer.toString(j));
                    answer = List.of(i, j);
                    break;
                }
            }
            if (!answer.isEmpty()) break;
        }
        emit("done", "result", listText(answer));
        System.out.println("{\"result\":\"" + listText(answer) + "\"}");
    }
}
