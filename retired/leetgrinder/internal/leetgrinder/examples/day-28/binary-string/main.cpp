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

string binary(int n, int depth) {
    emit("call", {{"n", to_string(n)}, {"depth", to_string(depth)}, {"result", "-"}});
    if (n < 2) {
        emit("base", {{"n", to_string(n)}, {"depth", to_string(depth)}, {"result", to_string(n)}});
        return to_string(n);
    }
    string text = binary(n / 2, depth + 1) + to_string(n % 2);
    emit("return", {{"n", to_string(n)}, {"depth", to_string(depth)}, {"result", text}});
    return text;
}

string solve(int n) { return binary(n, 0); }

int main() {
    int n;
    cin >> n;
    result(solve(n));
}
