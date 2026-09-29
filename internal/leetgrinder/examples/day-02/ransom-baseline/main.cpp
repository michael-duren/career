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

string usedState(const vector<bool> &used) {
  string out = "[";
  for (int i = 0; i < (int)used.size(); ++i)
    if (used[i]) {
      if (out.size() > 1)
        out += ", ";
      out += s(i);
    }
  return out + "]";
}
void solve(const string &note, const string &magazine) {
  bool ok = true;
  vector<bool> used(magazine.size(), false);
  emit("start", {{"i", "-1"},
                 {"letter", "-"},
                 {"position", "-1"},
                 {"state", "[]"},
                 {"ok", "true"}});
  for (int i = 0; i < (int)note.size(); ++i) {
    bool found = false;
    for (int j = 0; j < (int)magazine.size(); ++j) {
      emit("try", {{"i", s(i)},
                   {"letter", string(1, note[i])},
                   {"position", s(j)},
                   {"state", usedState(used)},
                   {"ok", "true"}});
      if (magazine[j] == note[i] && !used[j]) {
        used[j] = true;
        found = true;
        emit("take", {{"i", s(i)},
                      {"letter", string(1, note[i])},
                      {"position", s(j)},
                      {"state", usedState(used)},
                      {"ok", "true"}});
        break;
      }
    }
    if (!found) {
      ok = false;
      emit("missing", {{"i", s(i)},
                       {"letter", string(1, note[i])},
                       {"position", "-1"},
                       {"state", usedState(used)},
                       {"ok", "false"}});
      break;
    }
  }
  emit("done", {{"i", s(note.size())},
                {"letter", "-"},
                {"position", "-1"},
                {"state", usedState(used)},
                {"ok", ok ? "true" : "false"}});
  result(ok ? "true" : "false");
}
int main() {
  string note, magazine;
  cin >> note >> magazine;
  solve(note, magazine);
}
