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

int solve(const vector<int>& values, int target) {
    int lo = 0, hi = (int)values.size() - 1;
    int count = 0;
    emit("start", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"count", to_string(count)}});
    while (lo < hi) {
        if (values[lo] + values[hi] <= target) {
            count += hi - lo;
            ++lo;
            emit("count", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"count", to_string(count)}});
        } else {
            --hi;
            emit("too-big", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"count", to_string(count)}});
        }
    }

    emit("done", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"count", to_string(count)}});
    return count;
}

int main() {
    int n, target;
    cin >> n >> target;
    vector<int> values(n);
    for (int& value : values) cin >> value;
    result(to_string(solve(values, target)));
}
