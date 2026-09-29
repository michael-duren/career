#include <iostream>
#include <string>
#include <vector>

using Field = std::pair<std::string, std::string>;

void emit(const std::string& event, const std::vector<Field>& fields) {
    std::cout << "{\"event\":\"" << event << "\",\"variables\":[";
    for (size_t i = 0; i < fields.size(); ++i) {
        if (i) std::cout << ',';
        std::cout << "{\"name\":\"" << fields[i].first
                  << "\",\"value\":\"" << fields[i].second << "\"}";
    }
    std::cout << "]}\n";
}

std::string shown(const int counts[60]) {
    std::string text = "{";
    for (int r = 0; r < 60; ++r) {
        if (!counts[r]) continue;
        if (text.size() > 1) text += ',';
        text += std::to_string(r) + ':' + std::to_string(counts[r]);
    }
    return text + '}';
}

std::string listText(const std::vector<int>& values) {
    std::string text = "[";
    for (size_t i = 0; i < values.size(); ++i) {
        if (i) text += ',';
        text += std::to_string(values[i]);
    }
    return text + ']';
}

int main() {
    int size;
    if (!(std::cin >> size)) return 1;
    std::vector<int> songs(size);
    for (int& duration : songs) std::cin >> duration;
    int counts[60] = {};
    int pairs = 0;
    emit("start", {{"songs", listText(songs)}, {"counts", shown(counts)}, {"pairs", std::to_string(pairs)}});
    for (int i = 0; i < size; ++i) {
        int duration = songs[i];
        int remainder = duration % 60;
        int need = (60 - remainder) % 60;
        int matches = counts[need];
        emit("lookup", {{"i", std::to_string(i)}, {"duration", std::to_string(duration)}, {"remainder", std::to_string(remainder)}, {"need", std::to_string(need)}, {"matches", std::to_string(matches)}, {"pairs", std::to_string(pairs)}, {"counts", shown(counts)}});
        pairs += matches;
        emit("count", {{"i", std::to_string(i)}, {"need", std::to_string(need)}, {"matches", std::to_string(matches)}, {"pairs", std::to_string(pairs)}, {"counts", shown(counts)}});
        ++counts[remainder];
        emit("store", {{"i", std::to_string(i)}, {"remainder", std::to_string(remainder)}, {"pairs", std::to_string(pairs)}, {"counts", shown(counts)}});
    }
    emit("done", {{"pairs", std::to_string(pairs)}, {"counts", shown(counts)}});
    std::cout << "{\"result\":\"" << pairs << "\"}\n";
}
