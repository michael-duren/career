package leetgrinder

import (
	"strings"
	"testing"
)

func TestCatalogSeed(t *testing.T) {
	seed, err := CatalogSeed()
	if err != nil {
		t.Fatal(err)
	}
	if len(seed) != 300 {
		t.Fatalf("seed has %d problems, want the 300 curriculum problems", len(seed))
	}
	seen := map[string]bool{}
	numbers := map[int]string{}
	for _, p := range seed {
		if !ValidSlug(p.Slug) || seen[p.Slug] {
			t.Errorf("bad or duplicate slug %q", p.Slug)
		}
		seen[p.Slug] = true
		if other, dup := numbers[p.Number]; dup || p.Number < 1 || p.Number > MaxNumber {
			t.Errorf("%s: number %d (also %s)", p.Slug, p.Number, other)
		}
		numbers[p.Number] = p.Slug
		m := ProblemMetadata{Number: p.Number, Title: p.Title, Difficulty: p.Difficulty}
		if m.Normalize() != nil || p.Title == "" || p.Difficulty == "" || p.CurriculumTopic == "" {
			t.Errorf("%s: invalid metadata %+v", p.Slug, p)
		}
		for _, c := range []string{p.OptimalTime, p.OptimalSpace} {
			if v, err := NormalizeComplexity(c); err != nil || v != c || c == "" {
				t.Errorf("%s: optimal %q is not canonical", p.Slug, c)
			}
		}
		if len([]rune(p.OptimalNote)) > 400 {
			t.Errorf("%s: note over 400 characters", p.Slug)
		}
	}
	if two := seed[0]; two.Slug != "two-sum" || two.Number != 1 || two.OptimalTime != "O(n)" {
		t.Fatalf("first seed row %+v", two)
	}
}

func TestProblemMetadataNormalize(t *testing.T) {
	ok := ProblemMetadata{Number: 1, Title: "  Two Sum ", Difficulty: "Easy", Topics: []TopicTag{{"array", "Array"}, {"hash-table", "Hash Table"}, {"array", "Array"}}}
	if err := ok.Normalize(); err != nil || ok.Title != "Two Sum" || len(ok.Topics) != 2 || strings.Join(ok.TopicSlugs(), ",") != "array,hash-table" {
		t.Fatalf("normalize: %v %+v", err, ok)
	}
	if err := (&ProblemMetadata{}).Normalize(); err != nil {
		t.Fatalf("empty metadata rejected: %v", err)
	}
	many := make([]TopicTag, MaxTopics+1)
	for i := range many {
		many[i] = TopicTag{Slug: "t" + strings.Repeat("a", i+1), Name: "T"}
	}
	for name, m := range map[string]ProblemMetadata{
		"negative number":  {Number: -1},
		"huge number":      {Number: MaxNumber + 1},
		"difficulty":       {Difficulty: "easy"},
		"control title":    {Title: "Two\x00Sum"},
		"long title":       {Title: strings.Repeat("x", MaxTitleLength+1)},
		"bad topic slug":   {Topics: []TopicTag{{"Hash Table", "Hash Table"}}},
		"empty topic slug": {Topics: []TopicTag{{"", "Array"}}},
		"long topic name":  {Topics: []TopicTag{{"array", strings.Repeat("x", MaxTopicLength+1)}}},
		"too many topics":  {Topics: many},
	} {
		if m.Normalize() == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestNormalizeProblemRef(t *testing.T) {
	for in, want := range map[string]string{
		"two-sum":                                "two-sum",
		" Two-Sum ":                              "two-sum",
		"https://leetcode.com/problems/two-sum/": "two-sum",
		"https://leetcode.com/problems/two-sum/description/?envType=": "two-sum",
		"leetcode.com/problems/lru-cache":                             "lru-cache",
		"https://www.leetcode.com/problems/3sum/submissions/123/":     "3sum",
		"/problems/valid-anagram/":                                    "valid-anagram",
		"":                                                            "",
		"https://example.com/problems/two-sum/":                       "",
		"https://leetcode.com/contest/weekly-1/":                      "",
		"two sum":                                                     "",
		"two--sum":                                                    "",
		strings.Repeat("a", MaxSlugLength+1):                          "",
	} {
		got, ok := NormalizeProblemRef(in)
		if got != want || ok != (want != "") {
			t.Errorf("%q: %q %v, want %q", in, got, ok, want)
		}
	}
}

func TestTopicLabel(t *testing.T) {
	for slug, want := range map[string]string{"hash-table": "Hash Table", "dynamic-programming": "Dynamic Programming", "array": "Array", "": "Untagged"} {
		if got := TopicLabel(slug); got != want {
			t.Errorf("%q: %q", slug, got)
		}
	}
}

func TestProblemTopicLabelPrefersStoredName(t *testing.T) {
	p := Problem{Topics: []string{"depth-first-search", "hash-table"}, TopicNames: map[string]string{"depth-first-search": "Depth-First Search", "hash-table": ""}}
	if got := p.TopicLabel("depth-first-search"); got != "Depth-First Search" {
		t.Errorf("stored name: %q", got)
	}
	// An empty or missing name falls back to the title-cased slug.
	if got := p.TopicLabel("hash-table"); got != "Hash Table" {
		t.Errorf("empty name: %q", got)
	}
	if got := (Problem{}).TopicLabel("depth-first-search"); got != "Depth First Search" {
		t.Errorf("no names: %q", got)
	}
}
