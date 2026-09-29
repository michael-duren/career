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

string showSet(const set<char>& letters) {
    string out = "{";
    for (char letter : letters) out += (out.size() > 1 ? "," : "") + string(1, letter);
    return out + "}";
}

string solve(const vector<string>& grid) {
    int n = grid.size();
    vector<set<char>> rows(n), cols(n);
    emit("start", {{"r", "-"}, {"c", "-"}, {"letter", "-"}, {"row", "-"}, {"col", "-"}});
    for (int r = 0; r < n; ++r) {
        for (int c = 0; c < n; ++c) {
            char letter = grid[r][c];
            if (letter == '.') continue;
            if (rows[r].count(letter) || cols[c].count(letter)) {
                emit("clash", {{"r", to_string(r)}, {"c", to_string(c)}, {"letter", string(1, letter)}, {"row", showSet(rows[r])}, {"col", showSet(cols[c])}});
                return "false";
            }
            rows[r].insert(letter);
            cols[c].insert(letter);
            emit("add", {{"r", to_string(r)}, {"c", to_string(c)}, {"letter", string(1, letter)}, {"row", showSet(rows[r])}, {"col", showSet(cols[c])}});
        }
    }

    emit("done", {{"r", "-"}, {"c", "-"}, {"letter", "-"}, {"row", "-"}, {"col", "-"}});
    return "true";
}

int main() {
    int n;
    cin >> n;
    vector<string> grid(n);
    for (string& row : grid) cin >> row;
    result(solve(grid));
}
