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

#include <set>

vector<int> solve(const vector<int>& first, const vector<int>& second) {
    set<int> stops;
    int position = 0;
    for (size_t k = 0; k < first.size(); ++k) {
        position += first[k];
        stops.insert(position);
        emit("stop", {{"route", "A"}, {"length", to_string(first[k])}, {"position", to_string(position)}, {"shared", "[]"}});
    }
    vector<int> shared;
    position = 0;
    for (size_t k = 0; k < second.size(); ++k) {
        position += second[k];
        if (stops.count(position)) shared.push_back(position);
        emit("check", {{"route", "B"}, {"length", to_string(second[k])}, {"position", to_string(position)}, {"shared", show(shared)}});
    }

    emit("done", {{"route", "-"}, {"length", "-"}, {"position", "-"}, {"shared", show(shared)}});
    return shared;
}

vector<int> readLine() {
    string line;
    getline(cin, line);
    vector<int> values;
    size_t start = 0;
    while (start < line.size()) {
        size_t end = line.find(' ', start);
        if (end == string::npos) end = line.size();
        if (end > start) values.push_back(stoi(line.substr(start, end - start)));
        start = end + 1;
    }
    return values;
}

int main() {
    vector<int> first = readLine();
    vector<int> second = readLine();
    result(show(solve(first, second)));
}
