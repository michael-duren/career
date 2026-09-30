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

int solve(const vector<vector<int>>& grid, int limit) {
    int r = grid.size() - 1, c = 0;
    int count = 0;
    emit("start", {{"r", to_string(r)}, {"c", to_string(c)}, {"count", to_string(count)}});
    while (r >= 0 && c < (int)grid[0].size()) {
        if (grid[r][c] <= limit) {
            count += r + 1;
            ++c;
            emit("take-column", {{"r", to_string(r)}, {"c", to_string(c)}, {"count", to_string(count)}});
        } else {
            --r;
            emit("go-up", {{"r", to_string(r)}, {"c", to_string(c)}, {"count", to_string(count)}});
        }
    }

    emit("done", {{"r", to_string(r)}, {"c", to_string(c)}, {"count", to_string(count)}});
    return count;
}

int main() {
    int rows, cols, limit;
    cin >> rows >> cols >> limit;
    vector<vector<int>> grid(rows, vector<int>(cols));
    for (auto& row : grid)
        for (int& value : row) cin >> value;
    result(to_string(solve(grid, limit)));
}
