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

#include <map>

string showCounts(const map<string, int>& counts) {
    string out = "{";
    for (const auto& [key, count] : counts)
        if (count) out += (out.size() > 1 ? "," : "") + key + ":" + to_string(count);
    return out + "}";
}

int solve(const vector<string>& socks) {
    map<string, int> waiting;
    int pairs = 0;
    emit("start", {{"sock", "-"}, {"waiting", showCounts(waiting)}, {"pairs", to_string(pairs)}});
    for (const string& sock : socks) {
        ++waiting[sock];
        if (waiting[sock] == 2) {
            waiting[sock] = 0;
            ++pairs;
            emit("pair", {{"sock", sock}, {"waiting", showCounts(waiting)}, {"pairs", to_string(pairs)}});
        } else {
            emit("wait", {{"sock", sock}, {"waiting", showCounts(waiting)}, {"pairs", to_string(pairs)}});
        }
    }

    emit("done", {{"sock", "-"}, {"waiting", showCounts(waiting)}, {"pairs", to_string(pairs)}});
    return pairs;
}

int main() {
    int n;
    cin >> n;
    vector<string> socks(n);
    for (string& sock : socks) cin >> sock;
    result(to_string(solve(socks)));
}
