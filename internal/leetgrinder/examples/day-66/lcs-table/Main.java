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
        String answer=Integer.toString(dp[a.length()][b.length()]);
        emit("done","answer",answer);
        System.out.println("{\"result\":\""+answer+"\"}");
    }
}
