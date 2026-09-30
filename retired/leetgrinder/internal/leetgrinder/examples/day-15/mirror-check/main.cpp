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

string solve(const vector<int>& values, int target) {
    int lo = 0, hi = (int)values.size() - 1;
    emit("start", {{"lo", to_string(lo)}, {"hi", to_string(hi)}});
    while (lo < hi) {
        if (values[lo] != values[hi]) {
            emit("mismatch", {{"lo", to_string(lo)}, {"hi", to_string(hi)}});
            return "false";
        }
        ++lo;
        --hi;
        emit("match", {{"lo", to_string(lo)}, {"hi", to_string(hi)}});
    }

    emit("done", {{"lo", to_string(lo)}, {"hi", to_string(hi)}});
    return "true";
}

int main() {
    int n, target;
    cin >> n >> target;
    vector<int> values(n);
    for (int& value : values) cin >> value;
    result(solve(values, target));
}
