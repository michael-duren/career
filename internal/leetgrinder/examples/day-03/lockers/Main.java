import java.util.*;

public class Main {
    static void emit(String event, String... pairs) {
        StringBuilder out = new StringBuilder("{\"event\":\"" + event + "\",\"variables\":[");
        for (int k = 0; k < pairs.length; k += 2) {
            if (k > 0) out.append(',');
            out.append("{\"name\":\"").append(pairs[k]).append("\",\"value\":\"").append(pairs[k + 1]).append("\"}");
        }
        System.out.println(out.append("]}"));
    }

    static void result(String answer) { System.out.println("{\"result\":\"" + answer + "\"}"); }

    static String str(int value) { return Integer.toString(value); }

    static String show(int[] values) {
        StringBuilder out = new StringBuilder("[");
        for (int k = 0; k < values.length; k++) out.append(k > 0 ? "," : "").append(values[k]);
        return out.append("]").toString();
    }

    static String showMap(Map<String, String> mapping) {
        return mapping.toString().replace("=", ":").replace(", ", ",");
    }

    static String solve(String[][] pairs) {
        Map<String, String> lockerOf = new TreeMap<>(), studentOf = new TreeMap<>();
        emit("start", "student", "-", "locker", "-", "lockerOf", showMap(lockerOf), "studentOf", showMap(studentOf));
        for (String[] pair : pairs) {
            String student = pair[0], locker = pair[1];
            if (lockerOf.containsKey(student) && !lockerOf.get(student).equals(locker)) {
                emit("student-conflict", "student", student, "locker", locker, "lockerOf", showMap(lockerOf), "studentOf", showMap(studentOf));
                return "false";
            }
            if (studentOf.containsKey(locker) && !studentOf.get(locker).equals(student)) {
                emit("locker-conflict", "student", student, "locker", locker, "lockerOf", showMap(lockerOf), "studentOf", showMap(studentOf));
                return "false";
            }
            lockerOf.put(student, locker);
            studentOf.put(locker, student);
            emit("assign", "student", student, "locker", locker, "lockerOf", showMap(lockerOf), "studentOf", showMap(studentOf));
        }

        emit("done", "student", "-", "locker", "-", "lockerOf", showMap(lockerOf), "studentOf", showMap(studentOf));
        return "true";
    }

    public static void main(String[] args) {
        Scanner in = new Scanner(System.in);
        String[][] pairs = new String[in.nextInt()][2];
        for (String[] pair : pairs) { pair[0] = in.next(); pair[1] = in.next(); }
        result(solve(pairs));
    }
}
