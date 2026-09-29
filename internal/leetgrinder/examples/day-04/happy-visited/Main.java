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
    static int digitSquare(int value) {
        int total=0; while(value>0) {int digit=value%10;total+=digit*digit;value/=10;} return total;
    }
    public static void main(String[] args) {
        Scanner in=new Scanner(System.in);
        int value=in.nextInt(); Set<Integer> seen=new HashSet<>(); int step=0;
        emit("start", -1, Integer.toString(value), "seen=0", "pending");
        while(value!=1 && !seen.contains(value)) {
            seen.add(value); value=digitSquare(value);
            emit("step", step, Integer.toString(value), "seen="+seen.size(), "pending");
            step++;
        }
        String answer=value==1 ? "true" : "false";
        emit("done", step, Integer.toString(value), "seen="+seen.size(), answer);
        result(answer);
    }
}
