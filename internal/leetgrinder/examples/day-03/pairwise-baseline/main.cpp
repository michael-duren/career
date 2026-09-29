#include <iostream>
#include <sstream>
#include <string>
#include <unordered_map>
#include <vector>
using namespace std;
void emit(const string& event, int i, const string& left, const string& right, int pairs, bool valid) {
    cout << "{\"event\":\"" << event << "\",\"variables\":[";
    cout << "{\"name\":\"i\",\"value\":\"" << i << "\"},";
    cout << "{\"name\":\"left\",\"value\":\"" << left << "\"},";
    cout << "{\"name\":\"right\",\"value\":\"" << right << "\"},";
    cout << "{\"name\":\"pairs\",\"value\":\"" << pairs << "\"},";
    cout << "{\"name\":\"valid\",\"value\":\"" << (valid ? "true" : "false") << "\"}]}\n";
}
void result(bool value) { cout << "{\"result\":\"" << (value ? "true" : "false") << "\"}\n"; }

int main() {
    string left, right; getline(cin,left); getline(cin,right);
    vector<string> a,b; for (char c:left) a.push_back(string(1,c)); istringstream in(right); string w; while(in>>w) b.push_back(w);
    int checks=0;
    emit("start", -1, "-", "-", checks, true);
    if (a.size()!=b.size()) { emit("done", (int)a.size(), "-", "-", checks, false); result(false); return 0; }
    for (int i=0;i<(int)a.size();++i) {
        for (int j=0;j<i;++j) {
            ++checks;
            if ((a[i]==a[j]) != (b[i]==b[j])) {
                emit("reject", i, "i="+to_string(i)+",j="+to_string(j)+" "+a[i]+"/"+a[j], "i="+to_string(i)+",j="+to_string(j)+" "+b[i]+"/"+b[j], checks, false);
                emit("done", (int)a.size(), "-", "-", checks, false);
                result(false); return 0;
            }
            emit("compare", i, "i="+to_string(i)+",j="+to_string(j)+" "+a[i]+"/"+a[j], "i="+to_string(i)+",j="+to_string(j)+" "+b[i]+"/"+b[j], checks, true);
        }
    }
    emit("done", (int)a.size(), "-", "-", checks, true);
    result(true);
}
