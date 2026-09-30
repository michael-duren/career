#include <algorithm>
#include <iostream>
#include <string>
#include <utility>
#include <vector>
using namespace std;

void emit(const string& event, initializer_list<pair<string,string>> values) {
    cout << "{\"event\":\"" << event << "\",\"variables\":[";
    bool first = true;
    for (const auto& [name, value] : values) {
        if (!first) cout << ',';
        first = false;
        cout << "{\"name\":\"" << name << "\",\"value\":\"" << value << "\"}";
    }
    cout << "]}" << endl;
}
string rowText(const vector<int>& row) {
    string out;
    for (int value : row) { if (!out.empty()) out += ','; out += to_string(value); }
    return out;
}
int main() {
    string a, b;
    getline(cin, a); getline(cin, b);
    emit("start", {{"a",a},{"b",b}});
    auto visit = [&](auto&& self, int i, int j) -> int {
        emit("call", {{"i",to_string(i)},{"j",to_string(j)}});
        if (i==0 || j==0) return 0;
        if (a[i-1]==b[j-1]) return 1+self(self,i-1,j-1);
        int skipA=self(self,i-1,j);
        int skipB=self(self,i,j-1);
        return max(skipA,skipB);
    };
    string answer=to_string(visit(visit,a.size(),b.size()));
    emit("done", {{"answer",answer}});
    cout << "{\"result\":\"" << answer << "\"}" << endl;
}
