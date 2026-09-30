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

int step(int x) { return (x * x + 1) % 10; }

int solve(int start) {
    int slow = step(start);
    int fast = step(step(start));
    emit("first-move", {{"slow", to_string(slow)}, {"fast", to_string(fast)}});
    while (slow != fast) {
        slow = step(slow);
        fast = step(step(fast));
        emit("move", {{"slow", to_string(slow)}, {"fast", to_string(fast)}});
    }

    emit("meet", {{"slow", to_string(slow)}, {"fast", to_string(fast)}});
    return slow;
}

int main() {
    int start;
    cin >> start;
    result(to_string(solve(start)));
}
