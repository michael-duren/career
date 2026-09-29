import java.util.ArrayList;
import java.util.List;
import java.util.Scanner;

public class Main {
    static String shown(int[] counts) {
        StringBuilder text = new StringBuilder("{");
        for (int r = 0; r < 60; r++) {
            if (counts[r] == 0) continue;
            if (text.length() > 1) text.append(',');
            text.append(r).append(':').append(counts[r]);
        }
        return text.append('}').toString();
    }

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
        for (int j = 0; j < fields.length; j += 2) {
            if (j > 0) text.append(',');
            text.append("{\"name\":\"").append(fields[j]).append("\",\"value\":\"").append(fields[j + 1]).append("\"}");
        }
        System.out.println(text.append("]}"));
    }

    public static void main(String[] args) {
        Scanner input = new Scanner(System.in);
        int size = input.nextInt();
        List<Integer> songs = new ArrayList<>();
        for (int i = 0; i < size; i++) songs.add(input.nextInt());
        int[] counts = new int[60];
        int pairs = 0;
        emit("start", "songs", listText(songs), "counts", shown(counts), "pairs", Integer.toString(pairs));
        for (int i = 0; i < size; i++) {
            int duration = songs.get(i);
            int remainder = duration % 60;
            int need = (60 - remainder) % 60;
            int matches = counts[need];
            emit("lookup", "i", Integer.toString(i), "duration", Integer.toString(duration), "remainder", Integer.toString(remainder), "need", Integer.toString(need), "matches", Integer.toString(matches), "pairs", Integer.toString(pairs), "counts", shown(counts));
            pairs += matches;
            emit("count", "i", Integer.toString(i), "need", Integer.toString(need), "matches", Integer.toString(matches), "pairs", Integer.toString(pairs), "counts", shown(counts));
            counts[remainder]++;
            emit("store", "i", Integer.toString(i), "remainder", Integer.toString(remainder), "pairs", Integer.toString(pairs), "counts", shown(counts));
        }
        emit("done", "pairs", Integer.toString(pairs), "counts", shown(counts));
        System.out.println("{\"result\":\"" + pairs + "\"}");
    }
}
