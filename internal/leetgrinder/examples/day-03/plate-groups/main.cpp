#include <iostream>
#include <string>
#include <utility>
#include <vector>
using namespace std;

void emit(const string& event, const vector<pair<string, string>>& values) {
    cout << "{\"event\":\"" << event << "\",\"variables\":[";
    for (size_t k = 0; k < values.size(); ++k) {
        cout << (k ? "," : "") << "{\"name\":\"" << values[k].first << "\",\"value\":\"" << values[k].second << "\"}";
    }
    cout << "]}\n";
}

void result(const string& answer) { cout << "{\"result\":\"" << answer << "\"}\n"; }

string show(const vector<int>& values) {
    string out = "[";
    for (size_t k = 0; k < values.size(); ++k) out += (k ? "," : "") + to_string(values[k]);
    return out + "]";
}

#include <algorithm>
#include <cctype>
#include <map>

string showGroups(const vector<string>& keys, const map<string, vector<string>>& groups) {
    string out;
    for (const string& key : keys) {
        if (!out.empty()) out += "|";
        out += key + ":";
        const vector<string>& members = groups.at(key);
        for (size_t k = 0; k < members.size(); ++k) out += (k ? "," : "") + members[k];
    }
    return out;
}

string solve(const vector<string>& plates) {
    map<string, vector<string>> groups;
    vector<string> keys;
    emit("start", {{"plate", "-"}, {"key", "-"}, {"groups", showGroups(keys, groups)}});
    for (const string& plate : plates) {
        string key;
        for (char ch : plate) if (ch != '-') key += char(tolower(ch));
        if (!groups.count(key)) keys.push_back(key);
        groups[key].push_back(plate);
        emit("add", {{"plate", plate}, {"key", key}, {"groups", showGroups(keys, groups)}});
    }

    emit("done", {{"plate", "-"}, {"key", "-"}, {"groups", showGroups(keys, groups)}});
    return showGroups(keys, groups);
}

int main() {
    int n;
    cin >> n;
    vector<string> plates(n);
    for (string& plate : plates) cin >> plate;
    result(solve(plates));
}
