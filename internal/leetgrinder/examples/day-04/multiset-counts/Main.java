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
        int n=in.nextInt(); Map<Integer,Integer> remaining=new HashMap<>();
        for(int i=0;i<n;i++) {int value=in.nextInt();remaining.put(value,remaining.getOrDefault(value,0)+1);}
        int m=in.nextInt(); List<Integer> out=new ArrayList<>();
        emit("start", -1, "-", "remaining4="+remaining.getOrDefault(4,0), "empty");
        for(int i=0;i<m;i++) {
            int value=in.nextInt(); int count=remaining.getOrDefault(value,0);
            if(count>0) {remaining.put(value,count-1);out.add(value);}
            emit("step", i, Integer.toString(value), "remaining4="+remaining.getOrDefault(4,0), show(out));
        }
        String answer=show(out);
        emit("done", m, "-", "remaining4="+remaining.getOrDefault(4,0), answer);
        result(answer);
    }
}
