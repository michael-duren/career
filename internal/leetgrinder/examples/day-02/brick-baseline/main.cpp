#include <algorithm>
#include <iostream>
#include <map>
#include <string>
#include <vector>
using namespace std;

string quote(const string &s) {
  string out = "\"";
  for (char c : s) {
    if (c == '"' || c == '\\')
      out += '\\';
    out += c;
  }
  return out + "\"";
}
void emit(const string &event, initializer_list<pair<string, string>> fields) {
  cout << "{\"event\":" << quote(event) << ",\"variables\":[";
  bool first = true;
  for (auto [name, value] : fields) {
    if (!first)
      cout << ',';
    cout << "{\"name\":" << quote(name) << ",\"value\":" << quote(value) << '}';
    first = false;
  }
  cout << "]}\n";
}
void result(const string &value) {
  cout << "{\"result\":" << quote(value) << "}\n";
}
string s(int x) { return to_string(x); }

string edgeState(const map<int, int> &edges) {
  string out = "{";
  for (auto [x, n] : edges) {
    if (out.size() > 1)
      out += ",";
    out += s(x) + ":" + s(n);
  }
  return out + "}";
}
void solve(const vector<vector<int>> &rows) {
  int width = 0;
  for (int b : rows[0])
    width += b;
  int answer;
  int best = rows.size();
  emit("start", {{"x", "0"},
                 {"row", "-1"},
                 {"state", "outside"},
                 {"best", s(best)},
                 {"crossed", "0"}});
  for (int x = 1; x < width; ++x) {
    int crossed = 0;
    emit("probe", {{"x", s(x)},
                   {"row", "-1"},
                   {"state", "testing"},
                   {"best", s(best)},
                   {"crossed", s(crossed)}});
    for (int r = 0; r < (int)rows.size(); ++r) {
      int position = 0;
      bool hit = false;
      for (int j = 0; j + 1 < (int)rows[r].size(); ++j) {
        position += rows[r][j];
        if (position == x) {
          hit = true;
          break;
        }
      }
      if (!hit)
        ++crossed;
      emit("row", {{"x", s(x)},
                   {"row", s(r)},
                   {"state", hit ? "edge" : "brick"},
                   {"best", s(best)},
                   {"crossed", s(crossed)}});
    }
    best = min(best, crossed);
    emit("commit", {{"x", s(x)},
                    {"row", "-1"},
                    {"state", "completed"},
                    {"best", s(best)},
                    {"crossed", s(crossed)}});
  }
  emit("done", {{"x", s(width)},
                {"row", "-1"},
                {"state", "outside"},
                {"best", s(best)},
                {"crossed", s(best)}});
  answer = best;
  result(s(answer));
}
int main() {
  int n;
  cin >> n;
  vector<vector<int>> rows(n);
  for (auto &row : rows) {
    int k;
    cin >> k;
    row.resize(k);
    for (int &x : row)
      cin >> x;
  }
  solve(rows);
}
