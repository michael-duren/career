import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.util.ArrayList;
import java.util.List;

public class Main {
    static String a, b;
    static void emit(String event, String... fields) {
        StringBuilder s=new StringBuilder("{\"event\":\"").append(event).append("\",\"variables\":[");
        for (int k=0;k<fields.length;k+=2) {
            if (k>0) s.append(',');
            s.append("{\"name\":\"").append(fields[k]).append("\",\"value\":\"").append(fields[k+1]).append("\"}");
        }
        System.out.println(s.append("]}"));
    }
    static String rowText(int[] row) {
        StringBuilder s=new StringBuilder();
        for (int value: row) { if (s.length()>0) s.append(','); s.append(value); }
        return s.toString();
    }
    static int visit(int i,int j) {
        emit("call","i",Integer.toString(i),"j",Integer.toString(j));
        if (i==0 || j==0) return 0;
        if (a.charAt(i-1)==b.charAt(j-1)) return 1+visit(i-1,j-1);
        return Math.max(visit(i-1,j),visit(i,j-1));
    }

    public static void main(String[] args) throws Exception {
        BufferedReader in=new BufferedReader(new InputStreamReader(System.in));
        a=in.readLine(); b=in.readLine();
        emit("start","a",a,"b",b);
        String answer=Integer.toString(visit(a.length(),b.length()));
        emit("done","answer",answer);
        System.out.println("{\"result\":\""+answer+"\"}");
    }
}
