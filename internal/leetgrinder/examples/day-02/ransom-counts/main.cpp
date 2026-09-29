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

string countState(const vector<int> &counts) {
  string out = "{";
  for (int i = 0; i < 26; ++i)
    if (counts[i]) {
      if (out.size() > 1)
        out += ",";
      out += char('a' + i);
      out += ":" + s(counts[i]);
    }
  return out + "}";
}
void solve(const string &note, const string &magazine) {
  bool ok = true;
  vector<int> counts(26, 0);
  emit("start", {{"i", "-1"},
                 {"letter", "-"},
                 {"position", "-1"},
                 {"state", "{}"},
                 {"ok", "true"}});
  for (int j = 0; j < (int)magazine.size(); ++j) {
    counts[magazine[j] - 'a']++;
    emit("supply", {{"i", s(j)},
                    {"letter", string(1, magazine[j])},
                    {"position", s(j)},
                    {"state", countState(counts)},
                    {"ok", "true"}});
  }
  for (int i = 0; i < (int)note.size(); ++i) {
    counts[note[i] - 'a']--;
    ok = counts[note[i] - 'a'] >= 0;
    emit("request", {{"i", s(i)},
                     {"letter", string(1, note[i])},
                     {"position", "-1"},
                     {"state", countState(counts)},
                     {"ok", ok ? "true" : "false"}});
    if (!ok)
      break;
  }
  emit("done", {{"i", s(note.size())},
                {"letter", "-"},
                {"position", "-1"},
                {"state", countState(counts)},
                {"ok", ok ? "true" : "false"}});
  result(ok ? "true" : "false");
}
int main() {
  string note, magazine;
  cin >> note >> magazine;
  solve(note, magazine);
}
