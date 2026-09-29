import java.util.*;
public class Main {
    static String flat(List<Integer> a){
        StringJoiner j=new StringJoiner(",","[","]");for(int x:a)j.add(Integer.toString(x));return j.toString();
    }
    static String nested(List<List<Integer>> a){
        StringJoiner j=new StringJoiner(",","[","]");for(List<Integer> row:a)j.add(flat(row));return j.toString();
    }
    static void emit(String event,int depth,int visits,List<Integer> level,List<List<Integer>> result){
        System.out.println("{\"event\":\""+event+"\",\"variables\":["
            +"{\"name\":\"depth\",\"value\":\""+depth+"\"},"
            +"{\"name\":\"visits\",\"value\":\""+visits+"\"},"
            +"{\"name\":\"level\",\"value\":\""+flat(level)+"\"},"
            +"{\"name\":\"result\",\"value\":\""+nested(result)+"\"}]}");
    }
    static int height(int node,int[] left,int[] right){
        if(node==-1)return 0;
        return 1+Math.max(height(left[node],left,right),height(right[node],left,right));
    }
    static int collect(int node,int remaining,int[] value,int[] left,int[] right,List<Integer> level){
        if(node==-1)return 0;
        if(remaining==1){level.add(value[node]);return 1;}
        return 1+collect(left[node],remaining-1,value,left,right,level)
                +collect(right[node],remaining-1,value,left,right,level);
    }
    public static void main(String[] args){
        Scanner in=new Scanner(System.in);int n=in.nextInt();
        int[] value=new int[n],left=new int[n],right=new int[n];
        for(int i=0;i<n;i++){value[i]=in.nextInt();left[i]=in.nextInt();right[i]=in.nextInt();}
        List<List<Integer>> result=new ArrayList<>();int visits=0; // MODE: rescan
        emit("init",0,visits,List.of(),result);
        int maximum=n>0?height(0,left,right):0;
        emit("height",maximum,visits,List.of(),result);
        for(int depth=1;depth<=maximum;depth++){
            List<Integer> level=new ArrayList<>();
            emit("pass_start",depth,visits,level,result);
            visits+=collect(0,depth,value,left,right,level);
            result.add(level);
            emit("pass_end",depth,visits,level,result);
        }
        emit("done",maximum,visits,List.of(),result);
        System.out.println("{\"result\":\""+nested(result)+"\"}");
    }
}
