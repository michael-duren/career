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

string showCounts(const map<int, int>& counts) {
    string out = "{";
    for (auto it = counts.begin(); it != counts.end(); ++it)
        out += (it == counts.begin() ? "" : ",") + to_string(it->first) + ":" + to_string(it->second);
    return out + "}";
}

long long solve(const vector<int>& nums, int target) {
    map<int, int> counts;
    long long pairs = 0;
    emit("start", {{"i", "-"}, {"value", "-"}, {"need", "-"}, {"matches", "-"}, {"pairs", to_string(pairs)}, {"counts", showCounts(counts)}});
    for (int i = 0; i < (int)nums.size(); ++i) {
        int value = nums[i];
        int need = target - value;
        int matches = counts.count(need) ? counts[need] : 0;
        pairs += matches;
        emit("count", {{"i", to_string(i)}, {"value", to_string(value)}, {"need", to_string(need)}, {"matches", to_string(matches)}, {"pairs", to_string(pairs)}, {"counts", showCounts(counts)}});
        ++counts[value];
        emit("store", {{"i", to_string(i)}, {"value", to_string(value)}, {"need", to_string(need)}, {"matches", to_string(matches)}, {"pairs", to_string(pairs)}, {"counts", showCounts(counts)}});
    }

    emit("done", {{"i", "-"}, {"value", "-"}, {"need", "-"}, {"matches", "-"}, {"pairs", to_string(pairs)}, {"counts", showCounts(counts)}});
    return pairs;
}

int main() {
    int n, target;
    cin >> n >> target;
    vector<int> nums(n);
    for (int& value : nums) cin >> value;
    result(to_string(solve(nums, target)));
}
