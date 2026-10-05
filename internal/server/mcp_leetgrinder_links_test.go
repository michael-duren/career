package server

import (
	"testing"

	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestMCPLeetgrinderProblemLinks(t *testing.T) {
	for _, tc := range []struct{ slug, neetcode string }{
		{"two-sum", "https://neetcode.io/problems/two-integer-sum"},
		{"valid-sudoku", "https://neetcode.io/problems/valid-sudoku"},
		{"some-unmapped-problem", ""},
	} {
		t.Run(tc.slug, func(t *testing.T) {
			p := leetgrinder.Problem{Slug: tc.slug}
			results := []map[string]any{
				mcpLeetgrinderProblem(p),
				mcpLeetgrinderCard(leetgrinder.Today{Settings: leetgrinder.DefaultSettings()}, leetgrinder.Card{Problem: p}),
				todoResults([]leetgrinder.TodoItem{{Problem: p}})[0],
			}
			for _, result := range results {
				if result["url"] != "https://leetcode.com/problems/"+tc.slug+"/" {
					t.Errorf("LeetCode link changed: %v", result)
				}
				if tc.neetcode == "" {
					if _, ok := result["neetcodeUrl"]; ok {
						t.Errorf("unmapped problem has a NeetCode link: %v", result)
					}
				} else if result["neetcodeUrl"] != tc.neetcode {
					t.Errorf("NeetCode link = %v, want %s", result["neetcodeUrl"], tc.neetcode)
				}
			}
		})
	}
}
