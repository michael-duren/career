import java.util.*;
public class Main {
    static String flat(List<Integer> a) {
        StringJoiner j=new StringJoiner(",","[","]");
        for(int x:a)j.add(Integer.toString(x));
        return j.toString();
    }
    static String nested(List<List<Integer>> a) {
        StringJoiner j=new StringJoiner(",","[","]");
        for(List<Integer> row:a)j.add(flat(row));
        return j.toString();
    }
    static void emit(String event,int depth,int current,Deque<Integer> q,List<Integer> level,String result) {
        System.out.println("{\"event\":\""+event+"\",\"variables\":["
            +"{\"name\":\"depth\",\"value\":\""+depth+"\"},"
            +"{\"name\":\"current\",\"value\":\""+current+"\"},"
            +"{\"name\":\"queue\",\"value\":\""+flat(new ArrayList<>(q))+"\"},"
            +"{\"name\":\"level\",\"value\":\""+flat(level)+"\"},"
            +"{\"name\":\"result\",\"value\":\""+result+"\"}]}");
    }
    public static void main(String[] args){
        Scanner in=new Scanner(System.in);
        int n=in.nextInt();
        int[] value=new int[n],left=new int[n],right=new int[n];
        for(int i=0;i<n;i++){value[i]=in.nextInt();left[i]=in.nextInt();right[i]=in.nextInt();}
        Deque<Integer> q=new ArrayDeque<>();if(n>0)q.addLast(0);
        int depth=0;
        List<Integer> result=new ArrayList<>(); // MODE: right
        emit("init",depth,-1,q,List.of(),flat(result));
        while(!q.isEmpty()){
            depth++;int count=q.size();List<Integer> level=new ArrayList<>();
            emit("level_start",depth,-1,q,level,flat(result));
            for(int i=0;i<count;i++){
                int node=q.removeFirst();
                level.add(value[node]);
                if(left[node]!=-1)q.addLast(left[node]);
                if(right[node]!=-1)q.addLast(right[node]);
                emit("visit",depth,node,q,level,flat(result));
            }
            result.add(level.get(level.size()-1)); // COMMIT
            emit("level_end",depth,-1,q,level,flat(result));
        }
        emit("done",depth,-1,q,List.of(),flat(result));
        System.out.println("{\"result\":\""+flat(result)+"\"}");
    }
}
