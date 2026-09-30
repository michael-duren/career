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

vector<int> solve(const vector<int>& a, const vector<int>& b) {
    vector<int> merged;
    size_t i = 0, j = 0;
    emit("start", {{"i", to_string(i)}, {"j", to_string(j)}, {"merged", show(merged)}});
    while (i < a.size() || j < b.size()) {
        if (j == b.size() || (i < a.size() && a[i] <= b[j])) {
            merged.push_back(a[i]);
            ++i;
            emit("take-a", {{"i", to_string(i)}, {"j", to_string(j)}, {"merged", show(merged)}});
        } else {
            merged.push_back(b[j]);
            ++j;
            emit("take-b", {{"i", to_string(i)}, {"j", to_string(j)}, {"merged", show(merged)}});
        }
    }

    emit("done", {{"i", to_string(i)}, {"j", to_string(j)}, {"merged", show(merged)}});
    return merged;
}

int main() {
    int n, m;
    cin >> n;
    vector<int> a(n);
    for (int& value : a) cin >> value;
    cin >> m;
    vector<int> b(m);
    for (int& value : b) cin >> value;
    result(show(solve(a, b)));
}
