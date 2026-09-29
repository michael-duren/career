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

string showStack(const vector<string>& stack) {
    string out = "[";
    for (size_t k = 0; k < stack.size(); ++k) out += (k ? "," : "") + stack[k];
    return out + "]";
}

string solve(const vector<string>& tokens) {
    vector<string> text, redo;
    emit("start", {{"token", "-"}, {"stack", showStack(text)}, {"redo", showStack(redo)}});
    for (const string& token : tokens) {
        if (token == "undo") {
            if (!text.empty()) {
                redo.push_back(text.back());
                text.pop_back();
            }
            emit("undo", {{"token", token}, {"stack", showStack(text)}, {"redo", showStack(redo)}});
        } else if (token == "redo") {
            if (!redo.empty()) {
                text.push_back(redo.back());
                redo.pop_back();
            }
            emit("redo", {{"token", token}, {"stack", showStack(text)}, {"redo", showStack(redo)}});
        } else {
            text.push_back(token);
            redo.clear();
            emit("type", {{"token", token}, {"stack", showStack(text)}, {"redo", showStack(redo)}});
        }
    }

    emit("done", {{"token", "-"}, {"stack", showStack(text)}, {"redo", showStack(redo)}});
    string out;
    for (const string& letter : text) out += letter;
    return out.empty() ? "(empty)" : out;
}

int main() {
    vector<string> tokens;
    for (string token; cin >> token;) tokens.push_back(token);
    result(solve(tokens));
}
