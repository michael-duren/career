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

int solve(const vector<int>& values, long long target) {
    int lo = 0, hi = (int)values.size() - 1;
    emit("start", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", "-"}});
    while (lo <= hi) {
        int mid = lo + (hi - lo) / 2;
        if (values[mid] == target) {
            emit("found", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", to_string(mid)}});
            return mid;
        }
        if (values[mid] < target) {
            lo = mid + 1;
            emit("go-right", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", to_string(mid)}});
        } else {
            hi = mid - 1;
            emit("go-left", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", to_string(mid)}});
        }
    }

    emit("absent", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", "-"}});
    return -1;
}

int main() {
    int n;
    long long target;
    cin >> n >> target;
    vector<int> values(n);
    for (int& value : values) cin >> value;
    result(to_string(solve(values, target)));
}
