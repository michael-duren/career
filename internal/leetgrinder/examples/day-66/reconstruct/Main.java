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

    public static void main(String[] args) throws Exception {
        BufferedReader in=new BufferedReader(new InputStreamReader(System.in));
        a=in.readLine(); b=in.readLine();
        emit("start","a",a,"b",b);
        int[][] dp=new int[a.length()+1][b.length()+1];
        for (int i=1;i<=a.length();i++) {
            for (int j=1;j<=b.length();j++) {
                if (a.charAt(i-1)==b.charAt(j-1)) dp[i][j]=dp[i-1][j-1]+1;
                else dp[i][j]=Math.max(dp[i-1][j],dp[i][j-1]);
            }
            emit("row","i",Integer.toString(i),"row",rowText(dp[i]));
        }
        int i=a.length(),j=b.length(); StringBuilder out=new StringBuilder();
        while (i>0 && j>0) {
            int fromI=i,fromJ=j,upper=dp[i-1][j],left=dp[i][j-1];
            String choice;
            if (a.charAt(i-1)==b.charAt(j-1)) { choice="match"; out.append(a.charAt(i-1)); i--; j--; }
            else if (upper>=left) { choice="upper"; i--; }
            else { choice="left"; j--; }
            emit("walk","fromI",Integer.toString(fromI),"fromJ",Integer.toString(fromJ),"upper",Integer.toString(upper),"left",Integer.toString(left),"choice",choice,"i",Integer.toString(i),"j",Integer.toString(j),"out",out.toString());
        }
        String answer=out.reverse().toString();
        emit("done","answer",answer);
        System.out.println("{\"result\":\""+answer+"\"}");
    }
}
