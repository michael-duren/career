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

string showGrid(const vector<vector<int>>& grid) {
    string out = "[";
    for (size_t r = 0; r < grid.size(); ++r) out += (r ? "," : "") + show(grid[r]);
    return out + "]";
}

vector<vector<int>> solve(vector<vector<int>> grid) {
    int rows = grid.size(), cols = grid[0].size();
    vector<pair<int, int>> marks;
    for (int r = 0; r < rows; ++r) {
        for (int c = 0; c < cols; ++c) {
            if (grid[r][c] == 1) {
                marks.push_back({r, c});
                emit("mark", {{"r", to_string(r)}, {"c", to_string(c)}, {"marks", to_string(marks.size())}, {"grid", showGrid(grid)}});
            }
        }
    }

    for (auto [r, c] : marks) {
        if (r + 1 < rows) {
            grid[r + 1][c] = 1;
            emit("light", {{"r", to_string(r + 1)}, {"c", to_string(c)}, {"marks", to_string(marks.size())}, {"grid", showGrid(grid)}});
        }
    }

    emit("done", {{"r", "-"}, {"c", "-"}, {"marks", to_string(marks.size())}, {"grid", showGrid(grid)}});
    return grid;
}

int main() {
    int rows, cols;
    cin >> rows >> cols;
    vector<vector<int>> grid(rows, vector<int>(cols));
    for (auto& row : grid)
        for (int& value : row) cin >> value;
    result(showGrid(solve(grid)));
}
