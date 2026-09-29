#include <iostream>
#include <vector>
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
int main() {
    vector<string> board(9); for(string& row:board) cin>>row;
    struct Clue {int r,c; char digit;}; vector<Clue> clues;
    int step=0; string answer="true";
    emit("start", -1, "-", "checked=0", "pending");
    for(int r=0;r<9 && answer=="true";++r) for(int c=0;c<9;++c) {
        char digit=board[r][c]; if(digit=='.') continue;
        for(const auto& old:clues) if(old.digit==digit && (old.r==r || old.c==c || (old.r/3==r/3 && old.c/3==c/3))) {answer="false";break;}
        string current=string(1,digit)+"@"+to_string(r)+","+to_string(c);
        if(answer=="false") {emit("conflict", step, current, "checked="+to_string(clues.size()), answer);break;}
        clues.push_back({r,c,digit});
        emit("check", step, current, "checked="+to_string(clues.size()), "pending");
        ++step;
    }
    emit("done", step, "-", "checked="+to_string(clues.size()), answer);
    result(answer);
    return 0;
}
