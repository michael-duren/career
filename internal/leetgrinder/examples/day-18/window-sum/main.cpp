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
    int k = target;
    int total = 0;
    for (int right = 0; right < k; ++right) total += values[right];
    int best = total;
    emit("first", {{"right", to_string(k - 1)}, {"total", to_string(total)}, {"best", to_string(best)}});
    for (int right = k; right < (int)values.size(); ++right) {
        total += values[right] - values[right - k];
        best = max(best, total);
        emit("slide", {{"right", to_string(right)}, {"total", to_string(total)}, {"best", to_string(best)}});
    }

    emit("done", {{"right", "-"}, {"total", to_string(total)}, {"best", to_string(best)}});
    return best;
}

int main() {
    int n, target;
    cin >> n >> target;
    vector<int> values(n);
    for (int& value : values) cin >> value;
    result(to_string(solve(values, target)));
}
