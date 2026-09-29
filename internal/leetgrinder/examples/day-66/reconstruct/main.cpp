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
    vector<vector<int>> dp(a.size()+1, vector<int>(b.size()+1));
    for (int i=1; i<=static_cast<int>(a.size()); ++i) {
        for (int j=1; j<=static_cast<int>(b.size()); ++j) {
            if (a[i-1]==b[j-1]) dp[i][j]=dp[i-1][j-1]+1;
            else dp[i][j]=max(dp[i-1][j],dp[i][j-1]);
        }
        emit("row", {{"i",to_string(i)},{"row",rowText(dp[i])}});
    }
    int i=a.size(), j=b.size(); string out;
    while (i>0 && j>0) {
        int fromI=i, fromJ=j, upper=dp[i-1][j], left=dp[i][j-1];
        string choice;
        if (a[i-1]==b[j-1]) { choice="match"; out+=a[i-1]; --i; --j; }
        else if (upper>=left) { choice="upper"; --i; }
        else { choice="left"; --j; }
        emit("walk", {{"fromI",to_string(fromI)},{"fromJ",to_string(fromJ)},{"upper",to_string(upper)},{"left",to_string(left)},{"choice",choice},{"i",to_string(i)},{"j",to_string(j)},{"out",out}});
    }
    reverse(out.begin(),out.end()); string answer=out;
    emit("done", {{"answer",answer}});
    cout << "{\"result\":\"" << answer << "\"}" << endl;
}
