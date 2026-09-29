import java.util.*;
public class Main {
    static void emit(String event, int i, String left, String right, int pairs, boolean valid) {
        System.out.println("{\"event\":\"" + event + "\",\"variables\":["
            + "{\"name\":\"i\",\"value\":\"" + i + "\"},"
            + "{\"name\":\"left\",\"value\":\"" + left + "\"},"
            + "{\"name\":\"right\",\"value\":\"" + right + "\"},"
            + "{\"name\":\"pairs\",\"value\":\"" + pairs + "\"},"
            + "{\"name\":\"valid\",\"value\":\"" + valid + "\"}]}");
    }
    static void result(boolean value) { System.out.println("{\"result\":\"" + value + "\"}"); }

    public static void main(String[] args) {
        Scanner scanner=new Scanner(System.in); String left=scanner.nextLine(), right=scanner.hasNextLine()?scanner.nextLine():"";
        List<String> a=new ArrayList<>(), b=new ArrayList<>(); for(char c:left.toCharArray()) a.add(String.valueOf(c)); Scanner words=new Scanner(right); while(words.hasNext()) b.add(words.next());
        int checks=0;
        emit("start", -1, "-", "-", checks, true);
        if (a.size()!=b.size()) { emit("done", a.size(), "-", "-", checks, false); result(false); return; }
        for (int i=0;i<a.size();++i) {
            for (int j=0;j<i;++j) {
                ++checks;
                if (a.get(i).equals(a.get(j)) != b.get(i).equals(b.get(j))) {
                    emit("reject", i, "i="+i+",j="+j+" "+a.get(i)+"/"+a.get(j), "i="+i+",j="+j+" "+b.get(i)+"/"+b.get(j), checks, false);
                    emit("done", a.size(), "-", "-", checks, false);
                    result(false); return;
                }
                emit("compare", i, "i="+i+",j="+j+" "+a.get(i)+"/"+a.get(j), "i="+i+",j="+j+" "+b.get(i)+"/"+b.get(j), checks, true);
            }
        }
        emit("done", a.size(), "-", "-", checks, true);
        result(true);
    }
}
