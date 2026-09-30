#include <iostream>
#include <queue>
#include <string>
#include <vector>
using namespace std;
string flat(const vector<int>& a) {
    string s="[";
    for(size_t i=0;i<a.size();++i) {if(i)s+=",";s+=to_string(a[i]);}
    return s+"]";
}
string nested(const vector<vector<int>>& a) {
    string s="[";
    for(size_t i=0;i<a.size();++i) {if(i)s+=",";s+=flat(a[i]);}
    return s+"]";
}
vector<int> snapshot(queue<int> q) {
    vector<int> a;while(!q.empty()){a.push_back(q.front());q.pop();}return a;
}
void emit(string event,int depth,int current,queue<int> q,const vector<int>& level,string result) {
    cout<<"{\"event\":\""<<event<<"\",\"variables\":["
        <<"{\"name\":\"depth\",\"value\":\""<<depth<<"\"},"
        <<"{\"name\":\"current\",\"value\":\""<<current<<"\"},"
        <<"{\"name\":\"queue\",\"value\":\""<<flat(snapshot(q))<<"\"},"
        <<"{\"name\":\"level\",\"value\":\""<<flat(level)<<"\"},"
        <<"{\"name\":\"result\",\"value\":\""<<result<<"\"}]}"<<'\n';
}
int main(){
    int n;cin>>n;
    vector<int> value(n),left(n),right(n);
    for(int i=0;i<n;++i)cin>>value[i]>>left[i]>>right[i];
    queue<int> q;if(n)q.push(0);
    int depth=0;
    vector<int> result; // MODE: right
    emit("init",depth,-1,q,{},flat(result));
    while(!q.empty()){
        ++depth;int count=q.size();vector<int> level;
        emit("level_start",depth,-1,q,level,flat(result));
        for(int i=0;i<count;++i){
            int node=q.front();q.pop();
            level.push_back(value[node]);
            if(left[node]!=-1)q.push(left[node]);
            if(right[node]!=-1)q.push(right[node]);
            emit("visit",depth,node,q,level,flat(result));
        }
        result.push_back(level.back()); // COMMIT
        emit("level_end",depth,-1,q,level,flat(result));
    }
    emit("done",depth,-1,q,{},flat(result));
    cout<<"{\"result\":\""<<flat(result)<<"\"}\n";
}
