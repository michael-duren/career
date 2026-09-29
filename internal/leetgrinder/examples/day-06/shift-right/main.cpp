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

vector<int> solve(vector<int> values) {
    emit("start", {{"i", "-"}, {"values", show(values)}});
    for (int i = (int)values.size() - 1; i > 0; --i) {
        values[i] = values[i - 1];
        emit("copy", {{"i", to_string(i)}, {"values", show(values)}});
    }
    values[0] = 0;
    emit("fill", {{"i", "0"}, {"values", show(values)}});
    return values;
}

int main() {
    int n;
    cin >> n;
    vector<int> values(n);
    for (int& value : values) cin >> value;
    result(show(solve(values)));
}
