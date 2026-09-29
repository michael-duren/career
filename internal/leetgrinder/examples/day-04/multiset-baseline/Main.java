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
        int n=in.nextInt(); int[] first=new int[n]; for(int i=0;i<n;i++) first[i]=in.nextInt();
        int m=in.nextInt(); boolean[] used=new boolean[n]; int usedCount=0; List<Integer> out=new ArrayList<>();
        emit("start", -1, "-", "used=0", "empty");
        for(int i=0;i<m;i++) {
            int value=in.nextInt();
            for(int j=0;j<n;j++) if(!used[j] && first[j]==value) {used[j]=true;usedCount++;out.add(value);break;}
            emit("step", i, Integer.toString(value), "used="+usedCount, show(out));
        }
        String answer=show(out);
        emit("done", m, "-", "used="+usedCount, answer);
        result(answer);
    }
}
