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

#include <algorithm>

long long tripsBy(long long time, const vector<int>& rates) {
    long long trips = 0;
    for (int rate : rates) trips += time / rate;
    return trips;
}

long long solve(const vector<int>& rates, long long needed) {
    long long lo = 1, hi = (long long)*min_element(rates.begin(), rates.end()) * needed;
    emit("start", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", "-"}, {"trips", "-"}});
    while (lo < hi) {
        long long mid = lo + (hi - lo) / 2;
        long long trips = tripsBy(mid, rates);
        if (trips >= needed) {
            hi = mid;
            emit("enough", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", to_string(mid)}, {"trips", to_string(trips)}});
        } else {
            lo = mid + 1;
            emit("short", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", to_string(mid)}, {"trips", to_string(trips)}});
        }
    }

    emit("done", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"mid", "-"}, {"trips", to_string(tripsBy(lo, rates))}});
    return lo;
}

int main() {
    int n;
    long long needed;
    cin >> n >> needed;
    vector<int> rates(n);
    for (int& rate : rates) cin >> rate;
    result(to_string(solve(rates, needed)));
}
