#include <iostream>
#include <string>
#include <unordered_set>
using namespace std;

string escape(const string& value) {
    string out;
    for (char ch : value) {
        if (ch == '"' || ch == '\\') out += '\\';
        out += ch;
    }
    return out;
}
void emit(const string& event, int i, const string& current, const string& state, const string& answer) {
    cout << "{\"event\":\"" << escape(event) << "\",\"variables\":["
         << "{\"name\":\"i\",\"value\":\"" << i << "\"},"
         << "{\"name\":\"current\",\"value\":\"" << escape(current) << "\"},"
         << "{\"name\":\"state\",\"value\":\"" << escape(state) << "\"},"
         << "{\"name\":\"answer\",\"value\":\"" << escape(answer) << "\"}]}" << '\n';
}
void result(const string& answer) { cout << "{\"result\":\"" << escape(answer) << "\"}" << '\n'; }
int digitSquare(int value) {
    int total = 0;
    while (value) { int digit = value % 10; total += digit * digit; value /= 10; }
    return total;
}
int main() {
    int value; cin >> value; unordered_set<int> seen; int step=0;
    emit("start", -1, to_string(value), "seen=0", "pending");
    while(value!=1 && !seen.count(value)) {
        seen.insert(value); value=digitSquare(value);
        emit("step", step, to_string(value), "seen="+to_string(seen.size()), "pending");
        ++step;
    }
    string answer=value==1 ? "true" : "false";
    emit("done", step, to_string(value), "seen="+to_string(seen.size()), answer);
    result(answer);
    return 0;
}
