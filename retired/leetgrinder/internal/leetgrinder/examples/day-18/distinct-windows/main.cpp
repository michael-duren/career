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

#include <map>

int solve(const string& text, int k) {
    map<char, int> counts;
    int repeats = 0;
    int good = 0;
    emit("start", {{"right", "-"}, {"window", ""}, {"repeats", to_string(repeats)}, {"good", to_string(good)}});
    for (int right = 0; right < (int)text.size(); ++right) {
        char letter = text[right];
        if (++counts[letter] == 2) ++repeats;
        if (right >= k) {
            char old = text[right - k];
            if (--counts[old] == 1) --repeats;
        }
        if (right >= k - 1) {
            if (repeats == 0) ++good;
            emit("window", {{"right", to_string(right)}, {"window", text.substr(right - k + 1, k)}, {"repeats", to_string(repeats)}, {"good", to_string(good)}});
        }
    }

    emit("done", {{"right", "-"}, {"window", ""}, {"repeats", to_string(repeats)}, {"good", to_string(good)}});
    return good;
}

int main() {
    string text;
    int k;
    cin >> text >> k;
    result(to_string(solve(text, k)));
}
