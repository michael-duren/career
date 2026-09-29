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
    public static void main(String[] args) {
        Scanner in=new Scanner(System.in);
        String[] board=new String[9]; for(int i=0;i<9;i++) board[i]=in.next();
        List<Set<Character>> rows=new ArrayList<>(),cols=new ArrayList<>(),boxes=new ArrayList<>();
        for(int i=0;i<9;i++) {rows.add(new HashSet<>());cols.add(new HashSet<>());boxes.add(new HashSet<>());}
        int step=0; String answer="true";
        emit("start", -1, "-", "checked=0", "pending");
        for(int r=0;r<9 && answer.equals("true");r++) for(int c=0;c<9;c++) {
            char digit=board[r].charAt(c); if(digit=='.') continue;
            int box=(r/3)*3+c/3; String current=digit+"@"+r+","+c;
            if(rows.get(r).contains(digit)||cols.get(c).contains(digit)||boxes.get(box).contains(digit)) {
                answer="false";emit("conflict", step, current, "checked="+step, answer);break;
            }
            rows.get(r).add(digit);cols.get(c).add(digit);boxes.get(box).add(digit);step++;
            emit("check", step-1, current, "checked="+step, "pending");
        }
        emit("done", step, "-", "checked="+step, answer);
        result(answer);
    }
}
