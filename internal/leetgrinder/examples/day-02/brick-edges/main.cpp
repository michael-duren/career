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
  map<int, int> edges;
  int best = 0;
  emit("start", {{"x", "0"},
                 {"row", "-1"},
                 {"state", "{}"},
                 {"best", "0"},
                 {"crossed", s(rows.size())}});
  for (int r = 0; r < (int)rows.size(); ++r) {
    int x = 0;
    for (int j = 0; j + 1 < (int)rows[r].size(); ++j) {
      x += rows[r][j];
      ++edges[x];
      best = max(best, edges[x]);
      emit("edge", {{"x", s(x)},
                    {"row", s(r)},
                    {"state", edgeState(edges)},
                    {"best", s(best)},
                    {"crossed", s(rows.size() - best)}});
    }
  }
  emit("done", {{"x", s(width)},
                {"row", "-1"},
                {"state", edgeState(edges)},
                {"best", s(best)},
                {"crossed", s(rows.size() - best)}});
  answer = rows.size() - best;
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
