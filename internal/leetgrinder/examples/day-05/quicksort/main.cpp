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

void quicksort(vector<int>& values, int lo, int hi) {
    if (lo >= hi) return;
    int pivot = values[hi];
    int store = lo;
    emit("pivot", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"pivot", to_string(pivot)}, {"store", to_string(store)}, {"scan", "-"}, {"values", show(values)}});
    for (int scan = lo; scan < hi; ++scan) {
        if (values[scan] < pivot) {
            swap(values[store], values[scan]);
            ++store;
            emit("smaller", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"pivot", to_string(pivot)}, {"store", to_string(store)}, {"scan", to_string(scan)}, {"values", show(values)}});
        } else {
            emit("not-smaller", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"pivot", to_string(pivot)}, {"store", to_string(store)}, {"scan", to_string(scan)}, {"values", show(values)}});
        }
    }
    swap(values[store], values[hi]);
    emit("place", {{"lo", to_string(lo)}, {"hi", to_string(hi)}, {"pivot", to_string(pivot)}, {"store", to_string(store)}, {"scan", "-"}, {"values", show(values)}});
    quicksort(values, lo, store - 1);
    quicksort(values, store + 1, hi);
}

vector<int> solve(vector<int> values) {
    int last = (int)values.size() - 1;
    emit("start", {{"lo", "0"}, {"hi", to_string(last)}, {"pivot", "-"}, {"store", "-"}, {"scan", "-"}, {"values", show(values)}});
    quicksort(values, 0, last);

    emit("done", {{"lo", "0"}, {"hi", to_string(last)}, {"pivot", "-"}, {"store", "-"}, {"scan", "-"}, {"values", show(values)}});
    return values;
}

int main() {
    int n;
    cin >> n;
    vector<int> values(n);
    for (int& value : values) cin >> value;
    result(show(solve(values)));
}
