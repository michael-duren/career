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
    vector<string> stack;
    emit("start", {{"token", "-"}, {"stack", showStack(stack)}});
    for (const string& token : tokens) {
        if (token.rfind("</", 0) == 0) {
            string name = token.substr(2, token.size() - 3);
            if (stack.empty() || stack.back() != name) {
                emit("mismatch", {{"token", token}, {"stack", showStack(stack)}});
                return "false";
            }
            stack.pop_back();
            emit("close", {{"token", token}, {"stack", showStack(stack)}});
        } else {
            stack.push_back(token.substr(1, token.size() - 2));
            emit("open", {{"token", token}, {"stack", showStack(stack)}});
        }
    }

    emit("done", {{"token", "-"}, {"stack", showStack(stack)}});
    return stack.empty() ? "true" : "false";
}

int main() {
    vector<string> tokens;
    for (string token; cin >> token;) tokens.push_back(token);
    result(solve(tokens));
}
