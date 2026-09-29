#include <iostream>
#include <vector>
#include <string>
#include <unordered_map>
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
    int n,m; cin >> n; unordered_map<int,int> remaining;
    for(int i=0,x;i<n;++i) { cin>>x; ++remaining[x]; }
    cin >> m; vector<int> second(m); for(int& x:second) cin>>x;
    vector<int> out;
    emit("start", -1, "-", "remaining4="+to_string(remaining[4]), "empty");
    for(int i=0;i<m;++i) {
        int value=second[i];
        if(remaining[value]>0) { --remaining[value]; out.push_back(value); }
        emit("step", i, to_string(value), "remaining4="+to_string(remaining[4]), show(out));
    }
    string answer=show(out);
    emit("done", m, "-", "remaining4="+to_string(remaining[4]), answer);
    result(answer);
    return 0;
}
