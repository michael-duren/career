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

string showLetters(const int counts[26]) {
    string out = "{";
    for (int k = 0; k < 26; ++k)
        if (counts[k]) out += (out.size() > 1 ? "," : "") + string(1, char('a' + k)) + ":" + to_string(counts[k]);
    return out + "}";
}

string solve(const string& stock, const string& orders) {
    int counts[26] = {};
    for (char item : stock) ++counts[item - 'a'];
    emit("stock", {{"i", "-"}, {"item", "-"}, {"left", "-"}, {"counts", showLetters(counts)}});
    for (size_t i = 0; i < orders.size(); ++i) {
        char item = orders[i];
        int left = --counts[item - 'a'];
        emit("order", {{"i", to_string(i)}, {"item", string(1, item)}, {"left", to_string(left)}, {"counts", showLetters(counts)}});
        if (left < 0) return "short at " + to_string(i);
    }

    emit("done", {{"i", "-"}, {"item", "-"}, {"left", "-"}, {"counts", showLetters(counts)}});
    return "filled";
}

int main() {
    string stock, orders;
    getline(cin, stock);
    getline(cin, orders);
    result(solve(stock, orders));
}
