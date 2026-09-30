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

long long solve(long long n, long long target) {
    long long lo = 0, hi = n;
    emit("start", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", "-"}, {"value", "-"}});
    while (lo <= hi) {
        long long mid = lo + (hi - lo) / 2;
        long long value = mid * mid;
        if (value == target) {
            emit("found", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", to_string(mid)}, {"value", to_string(value)}});
            return mid;
        }
        if (value < target) {
            lo = mid + 1;
            emit("go-right", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", to_string(mid)}, {"value", to_string(value)}});
        } else {
            hi = mid - 1;
            emit("go-left", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", to_string(mid)}, {"value", to_string(value)}});
        }
    }

    emit("absent", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", "-"}, {"value", "-"}});
    return -1;
}

int main() {
    long long n, target;
    cin >> n >> target;
    result(to_string(solve(n, target)));
}
