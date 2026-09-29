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
        int value=in.nextInt(),slow=value,fast=value,step=0;
        emit("start", -1, Integer.toString(value), "slow="+slow+";fast="+fast, "pending");
        do {
            slow=digitSquare(slow); fast=digitSquare(digitSquare(fast));
            emit("step", step, Integer.toString(slow), "slow="+slow+";fast="+fast, "pending");
            step++;
        } while(slow!=1 && fast!=1 && slow!=fast);
        String answer=(slow==1 || fast==1) ? "true" : "false";
        emit("done", step, Integer.toString(slow), "slow="+slow+";fast="+fast, answer);
        result(answer);
    }
}
