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
  int candidate = 0, balance = 0;
  emit("start", {{"i", "-1"},
                 {"value", "-"},
                 {"candidate", "-"},
                 {"balance", "0"},
                 {"seen", "0"}});
  for (int i = 0; i < (int)values.size(); ++i) {
    int value = values[i];
    if (balance == 0) {
      candidate = value;
      balance = 1;
    } else if (candidate == value)
      ++balance;
    else
      --balance;
    emit("vote", {{"i", s(i)},
                  {"value", s(value)},
                  {"candidate", s(candidate)},
                  {"balance", s(balance)},
                  {"seen", "0"}});
  }
  int seen = 0;
  for (int i = 0; i < (int)values.size(); ++i) {
    int value = values[i];
    if (value == candidate)
      ++seen;
    emit("verify", {{"i", s(i)},
                    {"value", s(value)},
                    {"candidate", s(candidate)},
                    {"balance", s(balance)},
                    {"seen", s(seen)}});
  }
  if (seen > (int)values.size() / 2)
    answer = s(candidate);
  emit("done", {{"i", s(values.size())},
                {"value", "-"},
                {"candidate", s(candidate)},
                {"balance", s(balance)},
                {"seen", s(seen)}});
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
