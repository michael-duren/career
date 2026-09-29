import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.util.ArrayList;
import java.util.List;

public class Main {
    static String shown(int[] counts) {
        StringBuilder result = new StringBuilder("{");
        for (int i = 0; i < 26; i++) {
            if (counts[i] == 0) continue;
            if (result.length() > 1) result.append(",");
            result.append((char) ('a' + i)).append(":").append(counts[i]);
        }
        return result.append("}").toString();
    }

    static void emit(String event, String... fields) {
        List<String> variables = new ArrayList<>();
        for (int i = 0; i < fields.length; i += 2) {
            variables.add("{\"name\":\"" + fields[i] + "\",\"value\":\"" + fields[i + 1] + "\"}");
        }
        System.out.println("{\"event\":\"" + event + "\",\"variables\":[" + String.join(",", variables) + "]}");
    }

    public static void main(String[] args) throws Exception {
        BufferedReader input = new BufferedReader(new InputStreamReader(System.in));
        String first = input.readLine();
        String second = input.readLine();
        int[] counts = new int[26];
        emit("start", "first", first, "second", second, "counts", shown(counts));
        for (int i = 0; i < first.length(); i++) {
            char letter = first.charAt(i);
            counts[letter - 'a']++;
            emit("add", "i", Integer.toString(i), "letter", Character.toString(letter), "counts", shown(counts));
        }
        boolean answer = first.length() == second.length();
        if (answer) {
            for (int i = 0; i < second.length(); i++) {
                char letter = second.charAt(i);
                counts[letter - 'a']--;
                emit("remove", "i", Integer.toString(i), "letter", Character.toString(letter), "counts", shown(counts));
                if (counts[letter - 'a'] < 0) {
                    answer = false;
                    break;
                }
            }
        }
        emit("done", "counts", shown(counts), "anagram", Boolean.toString(answer));
        System.out.println("{\"result\":\"" + answer + "\"}");
    }
}
