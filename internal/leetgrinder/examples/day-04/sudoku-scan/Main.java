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
        List<int[]> clues=new ArrayList<>(); int step=0; String answer="true";
        emit("start", -1, "-", "checked=0", "pending");
        for(int r=0;r<9 && answer.equals("true");r++) for(int c=0;c<9;c++) {
            char digit=board[r].charAt(c); if(digit=='.') continue;
            for(int[] old:clues) if(old[2]==digit && (old[0]==r || old[1]==c || (old[0]/3==r/3 && old[1]/3==c/3))) {answer="false";break;}
            String current=digit+"@"+r+","+c;
            if(answer.equals("false")) {emit("conflict", step, current, "checked="+clues.size(), answer);break;}
            clues.add(new int[]{r,c,digit});
            emit("check", step, current, "checked="+clues.size(), "pending");
            step++;
        }
        emit("done", step, "-", "checked="+clues.size(), answer);
        result(answer);
    }
}
