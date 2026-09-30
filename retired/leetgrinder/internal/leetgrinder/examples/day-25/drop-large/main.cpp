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
    Node dummy{0, build(values)};
    Node* prev = &dummy;
    emit("start", {{"prev", "dummy"}, {"list", render(dummy.next)}});
    while (prev->next) {
        if (prev->next->value > target) {
            prev->next = prev->next->next;
            emit("remove", {{"prev", prev == &dummy ? "dummy" : to_string(prev->value)}, {"list", render(dummy.next)}});
        } else {
            prev = prev->next;
            emit("keep", {{"prev", to_string(prev->value)}, {"list", render(dummy.next)}});
        }
    }

    emit("done", {{"prev", "-"}, {"list", render(dummy.next)}});
    return render(dummy.next);
}

int main() {
    int n, target;
    cin >> n >> target;
    vector<int> values(n);
    for (int& value : values) cin >> value;
    result(solve(values, target));
}
