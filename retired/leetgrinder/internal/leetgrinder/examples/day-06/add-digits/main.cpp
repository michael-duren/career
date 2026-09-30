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

#include <algorithm>
#include <sstream>

vector<int> solve(const vector<int>& a, const vector<int>& b) {
    int i = (int)a.size() - 1, j = (int)b.size() - 1, carry = 0;
    vector<int> digits;
    emit("start", {{"i", to_string(i)}, {"j", to_string(j)}, {"carry", to_string(carry)}, {"digits", show(digits)}});
    while (i >= 0 || j >= 0 || carry) {
        int total = carry;
        if (i >= 0) total += a[i];
        if (j >= 0) total += b[j];
        digits.push_back(total % 10);
        carry = total / 10;
        emit("column", {{"i", to_string(i)}, {"j", to_string(j)}, {"carry", to_string(carry)}, {"digits", show(digits)}});
        --i;
        --j;
    }

    reverse(digits.begin(), digits.end());
    emit("done", {{"i", to_string(i)}, {"j", to_string(j)}, {"carry", to_string(carry)}, {"digits", show(digits)}});
    return digits;
}

vector<int> readDigits() {
    string line;
    getline(cin, line);
    istringstream in(line);
    vector<int> digits;
    for (int digit; in >> digit;) digits.push_back(digit);
    return digits;
}

int main() {
    vector<int> a = readDigits();
    vector<int> b = readDigits();
    result(show(solve(a, b)));
}
