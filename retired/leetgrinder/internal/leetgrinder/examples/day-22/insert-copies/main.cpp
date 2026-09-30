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

struct Node {
    int value;
    Node* next;
};

Node* build(const vector<int>& values) {
    Node* head = nullptr;
    for (auto it = values.rbegin(); it != values.rend(); ++it) head = new Node{*it, head};
    return head;
}

string render(Node* node) {
    string out;
    for (; node; node = node->next) out += (out.empty() ? "" : "->") + to_string(node->value);
    return out.empty() ? "empty" : out;
}

string solve(const vector<int>& values, int target) {
    Node* head = build(values);
    Node* cur = head;
    emit("start", {{"cur", cur ? to_string(cur->value) : "-"}, {"list", render(head)}});
    while (cur) {
        Node* copy = new Node{cur->value, cur->next};
        cur->next = copy;
        cur = copy->next;
        emit("insert", {{"cur", cur ? to_string(cur->value) : "-"}, {"list", render(head)}});
    }

    emit("done", {{"cur", "-"}, {"list", render(head)}});
    return render(head);
}

int main() {
    int n, target;
    cin >> n >> target;
    vector<int> values(n);
    for (int& value : values) cin >> value;
    result(solve(values, target));
}
