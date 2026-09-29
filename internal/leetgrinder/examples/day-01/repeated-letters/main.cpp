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

string solve(const string& word) {
    int counts[26] = {};
    emit("start", {{"i", "-"}, {"letter", "-"}, {"count", "-"}, {"repeated", ""}});
    for (size_t i = 0; i < word.size(); ++i) {
        char letter = word[i];
        ++counts[letter - 'a'];
        emit("count", {{"i", to_string(i)}, {"letter", string(1, letter)}, {"count", to_string(counts[letter - 'a'])}, {"repeated", ""}});
    }
    string repeated;
    for (int k = 0; k < 26; ++k) {
        if (counts[k] > 1) {
            repeated += char('a' + k);
            emit("repeat", {{"i", "-"}, {"letter", string(1, char('a' + k))}, {"count", to_string(counts[k])}, {"repeated", repeated}});
        }
    }

    emit("done", {{"i", "-"}, {"letter", "-"}, {"count", "-"}, {"repeated", repeated}});
    return repeated.empty() ? "none" : repeated;
}

int main() {
    string word;
    cin >> word;
    result(solve(word));
}
