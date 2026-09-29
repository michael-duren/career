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

string showSeen(const map<int, int>& firstSeen) {
    string out = "{";
    for (const auto& [hour, step] : firstSeen) out += (out.size() > 1 ? "," : "") + to_string(hour) + ":" + to_string(step);
    return out + "}";
}

string solve(const vector<int>& values, int target) {
    map<int, int> firstSeen = {{0, -1}};
    int hour = 0;
    emit("start", {{"step", "-"}, {"hour", to_string(hour)}, {"seen", showSeen(firstSeen)}});
    for (int step = 0; step < (int)values.size(); ++step) {
        hour = ((hour + values[step]) % target + target) % target;
        if (firstSeen.count(hour)) {
            emit("repeat", {{"step", to_string(step)}, {"hour", to_string(hour)}, {"seen", showSeen(firstSeen)}});
            return to_string(firstSeen[hour] + 1) + ".." + to_string(step);
        }
        firstSeen[hour] = step;
        emit("move", {{"step", to_string(step)}, {"hour", to_string(hour)}, {"seen", showSeen(firstSeen)}});
    }

    emit("done", {{"step", "-"}, {"hour", to_string(hour)}, {"seen", showSeen(firstSeen)}});
    return "none";
}

int main() {
    int n, target;
    cin >> n >> target;
    vector<int> values(n);
    for (int& value : values) cin >> value;
    result(solve(values, target));
}
