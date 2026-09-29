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

long long solve(long long n) {
    long long lo = 1, hi = n;
    emit("start", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", "-"}, {"blocks", "-"}, {"enough", "-"}});
    while (lo < hi) {
        long long mid = lo + (hi - lo) / 2;
        long long blocks = mid * (mid + 1) / 2;
        if (blocks >= n) {
            hi = mid;
            emit("yes", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", to_string(mid)}, {"blocks", to_string(blocks)}, {"enough", "true"}});
        } else {
            lo = mid + 1;
            emit("no", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", to_string(mid)}, {"blocks", to_string(blocks)}, {"enough", "false"}});
        }
    }

    emit("done", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", "-"}, {"blocks", to_string(lo * (lo + 1) / 2)}, {"enough", "true"}});
    return lo;
}

int main() {
    long long n;
    cin >> n;
    result(to_string(solve(n)));
}
