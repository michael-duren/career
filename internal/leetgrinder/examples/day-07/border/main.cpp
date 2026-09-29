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

vector<int> solve(const vector<vector<int>>& grid) {
    int rows = grid.size(), cols = grid[0].size();
    vector<int> order;
    emit("start", {{"side", "-"}, {"order", show(order)}});
    for (int c = 0; c < cols; ++c) order.push_back(grid[0][c]);
    emit("top", {{"side", "top"}, {"order", show(order)}});
    for (int r = 1; r < rows; ++r) order.push_back(grid[r][cols - 1]);
    emit("right", {{"side", "right"}, {"order", show(order)}});
    if (rows > 1) {
        for (int c = cols - 2; c >= 0; --c) order.push_back(grid[rows - 1][c]);
        emit("bottom", {{"side", "bottom"}, {"order", show(order)}});
    }
    if (cols > 1) {
        for (int r = rows - 2; r > 0; --r) order.push_back(grid[r][0]);
        emit("left", {{"side", "left"}, {"order", show(order)}});
    }

    emit("done", {{"side", "-"}, {"order", show(order)}});
    return order;
}

int main() {
    int rows, cols;
    cin >> rows >> cols;
    vector<vector<int>> grid(rows, vector<int>(cols));
    for (auto& row : grid)
        for (int& value : row) cin >> value;
    result(show(solve(grid)));
}
