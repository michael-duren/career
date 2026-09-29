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

long long solve(const vector<int>& values, long long limit) {
    long long count = 0;
    emit("start", {{"i", "-"}, {"j", "-"}, {"count", to_string(count)}});
    for (int i = 0; i < (int)values.size(); ++i) {
        int lo = i + 1, hi = values.size();
        while (lo < hi) {
            int mid = lo + (hi - lo) / 2;
            if (values[i] + values[mid] <= limit) lo = mid + 1;
            else hi = mid;
        }
        count += lo - i - 1;
        emit("row", {{"i", to_string(i)}, {"j", to_string(lo)}, {"count", to_string(count)}});
    }

    emit("done", {{"i", "-"}, {"j", "-"}, {"count", to_string(count)}});
    return count;
}

int main() {
    int n;
    long long target;
    cin >> n >> target;
    vector<int> values(n);
    for (int& value : values) cin >> value;
    result(to_string(solve(values, target)));
}
