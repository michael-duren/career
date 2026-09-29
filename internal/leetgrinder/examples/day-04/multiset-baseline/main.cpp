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
string show(const vector<int>& values) {
    if (values.empty()) return "empty";
    string out;
    for (int value : values) { if (!out.empty()) out += ","; out += to_string(value); }
    return out;
}
int main() {
    int n,m; cin >> n; vector<int> first(n); for(int& x:first) cin>>x;
    cin >> m; vector<int> second(m); for(int& x:second) cin>>x;
    vector<bool> used(n, false); int usedCount=0; vector<int> out;
    emit("start", -1, "-", "used=0", "empty");
    for(int i=0;i<m;++i) {
        int value=second[i];
        for(int j=0;j<n;++j) if(!used[j] && first[j]==value) { used[j]=true; ++usedCount; out.push_back(value); break; }
        emit("step", i, to_string(value), "used="+to_string(usedCount), show(out));
    }
    string answer=show(out);
    emit("done", m, "-", "used="+to_string(usedCount), answer);
    result(answer);
    return 0;
}
