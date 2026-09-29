#include <array>
#include <iostream>
#include <string>
#include <vector>
using namespace std;

struct Field { string name, value; };

string shown(const array<int, 26>& counts) {
    string result = "{";
    for (int i = 0; i < 26; ++i) {
        if (counts[i] == 0) continue;
        if (result.size() > 1) result += ",";
        result += char('a' + i);
        result += ":" + to_string(counts[i]);
    }
    return result + "}";
}

void emit(const string& event, const vector<Field>& fields) {
    cout << "{\"event\":\"" << event << "\",\"variables\":[";
    for (size_t i = 0; i < fields.size(); ++i) {
        if (i) cout << ",";
        cout << "{\"name\":\"" << fields[i].name
             << "\",\"value\":\"" << fields[i].value << "\"}";
    }
    cout << "]}" << '\n';
}

int main() {
    string first, second;
    getline(cin, first);
    getline(cin, second);
    array<int, 26> counts{};
    emit("start", {{"first", first}, {"second", second}, {"counts", shown(counts)}});
    for (size_t i = 0; i < first.size(); ++i) {
        char letter = first[i];
        ++counts[letter - 'a'];
        emit("add", {{"i", to_string(i)}, {"letter", string(1, letter)}, {"counts", shown(counts)}});
    }
    bool answer = first.size() == second.size();
    if (answer) {
        for (size_t i = 0; i < second.size(); ++i) {
            char letter = second[i];
            --counts[letter - 'a'];
            emit("remove", {{"i", to_string(i)}, {"letter", string(1, letter)}, {"counts", shown(counts)}});
            if (counts[letter - 'a'] < 0) {
                answer = false;
                break;
            }
        }
    }
    emit("done", {{"counts", shown(counts)}, {"anagram", answer ? "true" : "false"}});
    cout << "{\"result\":\"" << (answer ? "true" : "false") << "\"}" << '\n';
}
