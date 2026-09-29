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

string showMap(const map<string, string>& mapping) {
    string out = "{";
    for (auto it = mapping.begin(); it != mapping.end(); ++it)
        out += (it == mapping.begin() ? "" : ",") + it->first + ":" + it->second;
    return out + "}";
}

string solve(const vector<pair<string, string>>& pairs) {
    map<string, string> lockerOf, studentOf;
    emit("start", {{"student", "-"}, {"locker", "-"}, {"lockerOf", showMap(lockerOf)}, {"studentOf", showMap(studentOf)}});
    for (const auto& [student, locker] : pairs) {
        if (lockerOf.count(student) && lockerOf[student] != locker) {
            emit("student-conflict", {{"student", student}, {"locker", locker}, {"lockerOf", showMap(lockerOf)}, {"studentOf", showMap(studentOf)}});
            return "false";
        }
        if (studentOf.count(locker) && studentOf[locker] != student) {
            emit("locker-conflict", {{"student", student}, {"locker", locker}, {"lockerOf", showMap(lockerOf)}, {"studentOf", showMap(studentOf)}});
            return "false";
        }
        lockerOf[student] = locker;
        studentOf[locker] = student;
        emit("assign", {{"student", student}, {"locker", locker}, {"lockerOf", showMap(lockerOf)}, {"studentOf", showMap(studentOf)}});
    }

    emit("done", {{"student", "-"}, {"locker", "-"}, {"lockerOf", showMap(lockerOf)}, {"studentOf", showMap(studentOf)}});
    return "true";
}

int main() {
    int n;
    cin >> n;
    vector<pair<string, string>> pairs(n);
    for (auto& [student, locker] : pairs) cin >> student >> locker;
    result(solve(pairs));
}
