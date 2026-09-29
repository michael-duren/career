#include <iostream>
#include <string>
#include <vector>

using State = std::vector<std::pair<std::string, std::string>>;

void emit(const std::string& event, const State& state) {
    std::cout << "{\"event\":\"" << event << "\",\"variables\":[";
    for (size_t k = 0; k < state.size(); ++k) {
        if (k) std::cout << ',';
        std::cout << "{\"name\":\"" << state[k].first << "\",\"value\":\"" << state[k].second << "\"}";
    }
    std::cout << "]}\n";
}

std::string list_text(const std::vector<int>& values) {
    std::string text = "[";
    for (size_t k = 0; k < values.size(); ++k) {
        if (k) text += ',';
        text += std::to_string(values[k]);
    }
    return text + ']';
}

int main() {
    int count, target;
    if (!(std::cin >> count >> target)) return 1;
    std::vector<int> values(count);
    for (int& value : values) std::cin >> value;
    emit("start", {{"values", list_text(values)}, {"target", std::to_string(target)}});
    std::vector<int> answer;
    for (int i = 0; i < count; ++i) {
        emit("outer", {{"i", std::to_string(i)}, {"first", std::to_string(values[i])}});
        for (int j = i + 1; j < count; ++j) {
            long long total = static_cast<long long>(values[i]) + values[j];
            emit("compare", {{"i", std::to_string(i)}, {"j", std::to_string(j)}, {"left", std::to_string(values[i])}, {"right", std::to_string(values[j])}, {"sum", std::to_string(total)}});
            if (total == target) {
                emit("match", {{"i", std::to_string(i)}, {"j", std::to_string(j)}});
                answer = {i, j};
                break;
            }
        }
        if (!answer.empty()) break;
    }
    emit("done", {{"result", list_text(answer)}});
    std::cout << "{\"result\":\"" << list_text(answer) << "\"}\n";
}
