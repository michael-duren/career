#include <iostream>
#include <sstream>
#include <string>
#include <unordered_map>
#include <vector>
using namespace std;
void emit(const string& event, int i, const string& left, const string& right, int pairs, bool valid) {
    cout << "{\"event\":\"" << event << "\",\"variables\":[";
    cout << "{\"name\":\"i\",\"value\":\"" << i << "\"},";
    cout << "{\"name\":\"left\",\"value\":\"" << left << "\"},";
    cout << "{\"name\":\"right\",\"value\":\"" << right << "\"},";
    cout << "{\"name\":\"pairs\",\"value\":\"" << pairs << "\"},";
    cout << "{\"name\":\"valid\",\"value\":\"" << (valid ? "true" : "false") << "\"}]}\n";
}
void result(bool value) { cout << "{\"result\":\"" << (value ? "true" : "false") << "\"}\n"; }

int main() {
    string left, right; cin >> left >> right;
    vector<string> a, b; for (char c : left) a.push_back(string(1,c)); for (char c : right) b.push_back(string(1,c));
    unordered_map<string,string> forward, reverse;
    emit("start", -1, "-", "-", 0, true);
    if (a.size() != b.size()) { emit("done", (int)a.size(), "-", "-", 0, false); result(false); return 0; }
    for (int i=0; i<(int)a.size(); ++i) {
        const string &x=a[i], &y=b[i];
        if ((forward.count(x) && forward[x] != y) || (reverse.count(y) && reverse[y] != x)) {
            emit("reject", i, x, y, (int)forward.size(), false);
            emit("done", (int)a.size(), "-", "-", (int)forward.size(), false);
            result(false); return 0;
        }
        forward[x]=y; reverse[y]=x;
        emit("pair", i, x, y, (int)forward.size(), true);
    }
    emit("done", (int)a.size(), "-", "-", (int)forward.size(), true);
    result(true);
}
