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

vector<int> solve(vector<int> values, int target) {
    size_t write = 0;
    emit("start", {{"read", "-"}, {"write", to_string(write)}, {"values", show(values)}});
    for (size_t read = 0; read < values.size(); ++read) {
        if (values[read] != target) {
            values[write] = values[read];
            ++write;
            emit("keep", {{"read", to_string(read)}, {"write", to_string(write)}, {"values", show(values)}});
        } else {
            emit("drop", {{"read", to_string(read)}, {"write", to_string(write)}, {"values", show(values)}});
        }
    }

    emit("done", {{"read", "-"}, {"write", to_string(write)}, {"values", show(values)}});
    values.resize(write);
    return values;
}

int main() {
    int n, target;
    cin >> n >> target;
    vector<int> values(n);
    for (int& value : values) cin >> value;
    result(show(solve(values, target)));
}
