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

#include <set>

int step(int x) { return (x * x + 1) % 10; }

string showSet(const set<int>& seen) { return show(vector<int>(seen.begin(), seen.end())); }

int solve(int start) {
    set<int> seen;
    int x = start;
    emit("start", {{"x", to_string(x)}, {"seen", showSet(seen)}});
    while (!seen.count(x)) {
        seen.insert(x);
        x = step(x);
        emit("step", {{"x", to_string(x)}, {"seen", showSet(seen)}});
    }

    emit("repeat", {{"x", to_string(x)}, {"seen", showSet(seen)}});
    return x;
}

int main() {
    int start;
    cin >> start;
    result(to_string(solve(start)));
}
