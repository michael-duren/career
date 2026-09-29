#include <iostream>
#include <vector>
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
int main() {
    vector<string> board(9); for(string& row:board) cin>>row;
    vector<unordered_set<char>> rows(9),cols(9),boxes(9);
    int step=0; string answer="true";
    emit("start", -1, "-", "checked=0", "pending");
    for(int r=0;r<9 && answer=="true";++r) for(int c=0;c<9;++c) {
        char digit=board[r][c]; if(digit=='.') continue;
        int box=(r/3)*3+c/3;
        string current=string(1,digit)+"@"+to_string(r)+","+to_string(c);
        if(rows[r].count(digit)||cols[c].count(digit)||boxes[box].count(digit)) {
            answer="false"; emit("conflict", step, current, "checked="+to_string(step), answer); break;
        }
        rows[r].insert(digit); cols[c].insert(digit); boxes[box].insert(digit); ++step;
        emit("check", step-1, current, "checked="+to_string(step), "pending");
    }
    emit("done", step, "-", "checked="+to_string(step), answer);
    result(answer);
    return 0;
}
