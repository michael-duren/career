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

vector<long long> solve(const vector<int>& values, const vector<pair<int, int>>& queries) {
    vector<long long> prefix = {0};
    for (int value : values) prefix.push_back(prefix.back() + value);
    auto shown = [](const vector<long long>& list) {
        string out = "[";
        for (size_t k = 0; k < list.size(); ++k) out += (k ? "," : "") + to_string(list[k]);
        return out + "]";
    };
    emit("prefix", {{"l", "-"}, {"r", "-"}, {"answer", "-"}, {"prefix", shown(prefix)}});
    vector<long long> answers;
    for (auto [l, r] : queries) {
        answers.push_back(prefix[r + 1] - prefix[l]);
        emit("query", {{"l", to_string(l)}, {"r", to_string(r)}, {"answer", to_string(answers.back())}, {"prefix", shown(prefix)}});
    }
    return answers;
}

int main() {
    int n, q;
    cin >> n;
    vector<int> values(n);
    for (int& value : values) cin >> value;
    cin >> q;
    vector<pair<int, int>> queries(q);
    for (auto& [l, r] : queries) cin >> l >> r;
    vector<long long> answers = solve(values, queries);
    string out = "[";
    for (size_t k = 0; k < answers.size(); ++k) out += (k ? "," : "") + to_string(answers[k]);
    result(out + "]");
}
