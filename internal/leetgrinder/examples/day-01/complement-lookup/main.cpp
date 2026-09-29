#include <algorithm>
#include <iostream>
#include <map>
#include <string>
#include <vector>

using State = std::vector<std::pair<std::string, std::string>>;

void emit(const std::string& event, const State& state) {
    std::cout << "{\"event\":\"" << event << "\",\"variables\":[";
    for (size_t j = 0; j < state.size(); ++j) {
        if (j) std::cout << ',';
        std::cout << "{\"name\":\"" << state[j].first << "\",\"value\":\"" << state[j].second << "\"}";
    }
    std::cout << "]}\n";
}

std::string shown(const std::map<int, int>& seen) {
    std::string text = "{";
    for (const auto& [value, index] : seen) {
        if (text.size() > 1) text += ',';
        text += std::to_string(value) + ':' + std::to_string(index);
    }
    return text + '}';
}

std::string list_text(const std::vector<int>& values) {
    std::string text = "[";
    for (size_t j = 0; j < values.size(); ++j) {
        if (j) text += ',';
        text += std::to_string(values[j]);
    }
    return text + ']';
}

int main() {
    int count, target;
    if (!(std::cin >> count >> target)) return 1;
    std::vector<int> values(count);
    for (int& value : values) std::cin >> value;
    std::map<int, int> seen;
    emit("start", {{"values", list_text(values)}, {"target", std::to_string(target)}, {"seen", shown(seen)}});
    std::vector<int> answer;
    for (int i = 0; i < count; ++i) {
        int value = values[i];
        int need = target - value;
        emit("lookup", {{"i", std::to_string(i)}, {"value", std::to_string(value)}, {"need", std::to_string(need)}, {"seen", shown(seen)}});
        auto earlier = seen.find(need);
        if (earlier != seen.end()) {
            emit("match", {{"i", std::to_string(i)}, {"need", std::to_string(need)}, {"earlier", std::to_string(earlier->second)}, {"seen", shown(seen)}});
            answer = {earlier->second, i};
            break;
        }
        seen[value] = i;
        emit("insert", {{"i", std::to_string(i)}, {"seen", shown(seen)}});
    }
    emit("done", {{"result", list_text(answer)}});
    std::cout << "{\"result\":\"" << list_text(answer) << "\"}\n";
}
