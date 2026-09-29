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

void solve(const vector<int> &values) {
  string answer = "none";
  int lastSeen = 0, lastCandidate = 0;
  emit("start",
       {{"i", "-1"}, {"candidate", "-"}, {"tally", "0"}, {"answer", "none"}});
  for (int i = 0; i < (int)values.size(); ++i) {
    int candidate = values[i], seen = 0;
    for (int value : values)
      if (value == candidate)
        ++seen;
    lastSeen = seen;
    lastCandidate = candidate;
    emit("test", {{"i", s(i)},
                  {"candidate", s(candidate)},
                  {"tally", s(seen)},
                  {"answer", "none"}});
    if (seen > (int)values.size() / 2) {
      answer = s(candidate);
      break;
    }
  }
  emit("done", {{"i", s(values.size())},
                {"candidate", s(lastCandidate)},
                {"tally", s(lastSeen)},
                {"answer", answer}});
  result(answer);
}
int main() {
  int n;
  cin >> n;
  vector<int> values(n);
  for (int &x : values)
    cin >> x;
  solve(values);
}
