#include <algorithm>
#include <iostream>
#include <string>
#include <vector>
using namespace std;
string flat(const vector<int>& a){
    string s="[";for(size_t i=0;i<a.size();++i){if(i)s+=",";s+=to_string(a[i]);}return s+"]";
}
string nested(const vector<vector<int>>& a){
    string s="[";for(size_t i=0;i<a.size();++i){if(i)s+=",";s+=flat(a[i]);}return s+"]";
}
void emit(string event,int depth,int visits,const vector<int>& level,const vector<vector<int>>& result){
    cout<<"{\"event\":\""<<event<<"\",\"variables\":["
        <<"{\"name\":\"depth\",\"value\":\""<<depth<<"\"},"
        <<"{\"name\":\"visits\",\"value\":\""<<visits<<"\"},"
        <<"{\"name\":\"level\",\"value\":\""<<flat(level)<<"\"},"
        <<"{\"name\":\"result\",\"value\":\""<<nested(result)<<"\"}]}"<<'\n';
}
int height(int node,const vector<int>& left,const vector<int>& right){
    if(node==-1)return 0;
    return 1+max(height(left[node],left,right),height(right[node],left,right));
}
int collect(int node,int remaining,const vector<int>& value,const vector<int>& left,const vector<int>& right,vector<int>& level){
    if(node==-1)return 0;
    if(remaining==1){level.push_back(value[node]);return 1;}
    return 1+collect(left[node],remaining-1,value,left,right,level)
            +collect(right[node],remaining-1,value,left,right,level);
}
int main(){
    int n;cin>>n;vector<int> value(n),left(n),right(n);
    for(int i=0;i<n;++i)cin>>value[i]>>left[i]>>right[i];
    vector<vector<int>> result;int visits=0; // MODE: rescan
    emit("init",0,visits,{},result);
    int maximum=n?height(0,left,right):0;
    emit("height",maximum,visits,{},result);
    for(int depth=1;depth<=maximum;++depth){
        vector<int> level;
        emit("pass_start",depth,visits,level,result);
        visits+=collect(0,depth,value,left,right,level);
        result.push_back(level);
        emit("pass_end",depth,visits,level,result);
    }
    emit("done",maximum,visits,{},result);
    cout<<"{\"result\":\""<<nested(result)<<"\"}\n";
}
