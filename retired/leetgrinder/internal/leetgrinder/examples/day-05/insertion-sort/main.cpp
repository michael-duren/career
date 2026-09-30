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

vector<int> solve(vector<int> values) {
    emit("start", {{"i", "-"}, {"key", "-"}, {"j", "-"}, {"values", show(values)}});
    for (int i = 1; i < (int)values.size(); ++i) {
        int key = values[i];
        int j = i - 1;
        emit("take", {{"i", to_string(i)}, {"key", to_string(key)}, {"j", to_string(j)}, {"values", show(values)}});
        while (j >= 0 && values[j] > key) {
            values[j + 1] = values[j];
            emit("shift", {{"i", to_string(i)}, {"key", to_string(key)}, {"j", to_string(j)}, {"values", show(values)}});
            --j;
        }
        values[j + 1] = key;
        emit("place", {{"i", to_string(i)}, {"key", to_string(key)}, {"j", to_string(j)}, {"values", show(values)}});
    }

    emit("done", {{"i", to_string(values.size())}, {"key", "-"}, {"j", "-"}, {"values", show(values)}});
    return values;
}

int main() {
    int n;
    cin >> n;
    vector<int> values(n);
    for (int& value : values) cin >> value;
    result(show(solve(values)));
}
