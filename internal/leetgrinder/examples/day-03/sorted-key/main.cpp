#include <algorithm>
#include <iostream>
#include <string>
#include <unordered_map>
#include <vector>
using namespace std;
void emit(const string& event, int i, const string& word, const string& key, const string& groups) {
    cout << "{\"event\":\"" << event << "\",\"variables\":[";
    cout << "{\"name\":\"i\",\"value\":\"" << i << "\"},";
    cout << "{\"name\":\"word\",\"value\":\"" << word << "\"},";
    cout << "{\"name\":\"key\",\"value\":\"" << key << "\"},";
    cout << "{\"name\":\"groups\",\"value\":\"" << groups << "\"}]}\n";
}
void result(const string& value) { cout << "{\"result\":\"" << value << "\"}\n"; }
string render(const vector<vector<string>>& groups) {
    string out; for (const auto& group:groups) { if (!out.empty()) out+="|"; for (const string& word:group) { if (&word != &group[0]) out+=","; out+=word; } } return out;
}
int main() {
    int n; cin >> n; vector<string> words(n); for (string& w:words) cin >> w;
    unordered_map<string,int> keys; vector<vector<string>> groups;
    emit("start", -1, "-", "-", "-");
    for (int i=0;i<n;++i) {
        string word=words[i];
        string key=word; sort(key.begin(), key.end());
        if (!keys.count(key)) { keys[key]=(int)groups.size(); groups.push_back({}); }
        groups[keys[key]].push_back(word);
        emit("group", i, word, key, render(groups));
    }
    string answer=render(groups);
    emit("done", n, "-", "-", answer);
    result(answer);
}
