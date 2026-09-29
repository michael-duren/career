import java.util.*;
public class Main {
    static void emit(String event, int i, String word, String key, String groups) {
        System.out.println("{\"event\":\""+event+"\",\"variables\":["
            + "{\"name\":\"i\",\"value\":\""+i+"\"},"
            + "{\"name\":\"word\",\"value\":\""+word+"\"},"
            + "{\"name\":\"key\",\"value\":\""+key+"\"},"
            + "{\"name\":\"groups\",\"value\":\""+groups+"\"}]}");
    }
    static void result(String value) { System.out.println("{\"result\":\""+value+"\"}"); }
    static String render(List<List<String>> groups) { List<String> parts=new ArrayList<>(); for (List<String> group:groups) parts.add(String.join(",", group)); return String.join("|", parts); }
    public static void main(String[] args) {
        Scanner in=new Scanner(System.in); int n=in.nextInt(); List<String> words=new ArrayList<>(); for(int i=0;i<n;++i) words.add(in.next());
        Map<String,Integer> keys=new HashMap<>(); List<List<String>> groups=new ArrayList<>();
        emit("start", -1, "-", "-", "-");
        for (int i=0;i<n;++i) {
            String word=words.get(i);
            char[] chars=word.toCharArray(); Arrays.sort(chars); String key=new String(chars);
            if (!keys.containsKey(key)) { keys.put(key,groups.size()); groups.add(new ArrayList<>()); }
            groups.get(keys.get(key)).add(word);
            emit("group", i, word, key, render(groups));
        }
        String answer=render(groups);
        emit("done", n, "-", "-", answer);
        result(answer);
    }
}
