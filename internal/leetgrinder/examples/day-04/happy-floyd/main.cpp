#include <iostream>
#include <string>
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
    int value; cin >> value; int slow=value,fast=value,step=0;
    auto state=[&]() {return "slow="+to_string(slow)+";fast="+to_string(fast);};
    emit("start", -1, to_string(value), state(), "pending");
    do {
        slow=digitSquare(slow); fast=digitSquare(digitSquare(fast));
        emit("step", step, to_string(slow), state(), "pending");
        ++step;
    } while(slow!=1 && fast!=1 && slow!=fast);
    string answer=(slow==1 || fast==1) ? "true" : "false";
    emit("done", step, to_string(slow), state(), answer);
    result(answer);
    return 0;
}
