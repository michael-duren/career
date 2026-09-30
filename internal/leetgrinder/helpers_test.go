package leetgrinder

import "time"

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

// testProblems is a small catalog for tests.
var testProblems = map[string]Problem{
	"two-sum":            {Slug: "two-sum", Number: 1, Title: "Two Sum", Difficulty: "Easy", Topics: []string{"array", "hash-table"}, OptimalTime: "O(n)", OptimalSpace: "O(n)", OptimalSource: "curated", MetadataSource: "seed"},
	"binary-search":      {Slug: "binary-search", Number: 704, Title: "Binary Search", Difficulty: "Easy", Topics: []string{"array", "binary-search"}, OptimalTime: "O(log n)", OptimalSpace: "O(1)", OptimalSource: "curated", MetadataSource: "seed"},
	"isomorphic-strings": {Slug: "isomorphic-strings", Number: 205, Title: "Isomorphic Strings", Difficulty: "Easy", Topics: []string{"hash-table", "string"}, MetadataSource: "extension"},
	"ransom-note":        {Slug: "ransom-note", Number: 383, Title: "Ransom Note", Difficulty: "Easy", OptimalTime: "O(m + n)", OptimalSpace: "O(1)", OptimalSource: "model", MetadataSource: "leetcode"},
	"valid-anagram":      {Slug: "valid-anagram", Number: 242, Title: "Valid Anagram", Difficulty: "Easy", MetadataSource: "seed"},
}
