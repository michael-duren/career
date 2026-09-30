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
    Node* slow = head;
    Node* fast = head->next;
    while (fast->next) {
        slow = slow->next;
        fast = fast->next->next;
    }
    Node* second = slow->next;
    slow->next = nullptr;
    Node* prev = nullptr;
    while (second) {
        Node* following = second->next;
        second->next = prev;
        prev = second;
        second = following;
    }
    emit("halves", {{"first", render(head)}, {"second", render(prev)}, {"best", "-"}});
    int best = 0;
    for (Node *a = head, *b = prev; a; a = a->next, b = b->next) {
        best = max(best, a->value + b->value);
        emit("pair", {{"first", to_string(a->value)}, {"second", to_string(b->value)}, {"best", to_string(best)}});
    }

    emit("done", {{"first", "-"}, {"second", "-"}, {"best", to_string(best)}});
    return to_string(best);
}

int main() {
    int n, target;
    cin >> n >> target;
    vector<int> values(n);
    for (int& value : values) cin >> value;
    result(solve(values, target));
}
