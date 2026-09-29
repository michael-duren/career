import java.util.*;

public class Main {
    static String esc(String value) { return value.replace("\\", "\\\\").replace("\"", "\\\""); }
    static void emit(String event, int i, String current, String state, String answer) {
        System.out.println("{\"event\":\""+esc(event)+"\",\"variables\":["
            +"{\"name\":\"i\",\"value\":\""+i+"\"},"
            +"{\"name\":\"current\",\"value\":\""+esc(current)+"\"},"
            +"{\"name\":\"state\",\"value\":\""+esc(state)+"\"},"
            +"{\"name\":\"answer\",\"value\":\""+esc(answer)+"\"}]}");
    }
    static void result(String answer) { System.out.println("{\"result\":\""+esc(answer)+"\"}"); }
    static String show(List<Integer> values) {
        if (values.isEmpty()) return "empty";
        StringJoiner joiner=new StringJoiner(","); for(int value:values) joiner.add(Integer.toString(value)); return joiner.toString();
    }
    public static void main(String[] args) {
        Scanner in=new Scanner(System.in);
        int n=in.nextInt(); Set<Integer> source=new HashSet<>(); for(int i=0;i<n;i++) source.add(in.nextInt());
        int m=in.nextInt(); Set<Integer> emitted=new HashSet<>(); List<Integer> out=new ArrayList<>();
        emit("start", -1, "-", "emitted=empty", "empty");
        for(int i=0;i<m;i++) {
            int value=in.nextInt();
            if(source.contains(value) && emitted.add(value)) out.add(value);
            emit("step", i, Integer.toString(value), "emitted="+show(out), show(out));
        }
        String answer=show(out);
        emit("done", m, "-", "emitted="+show(out), answer);
        result(answer);
    }
}
