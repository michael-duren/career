package leetgrinder

import (
	"errors"
	"os"
	"regexp"
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
		// leetcode.cn uses the same slugs.
		"https://leetcode.cn/problems/two-sum/":                       "two-sum",
		"https://leetcode.cn/problems/two-sum/description/?envType=x": "two-sum",
		"https://www.leetcode.cn/problems/lru-cache/solutions/":       "lru-cache",
		"leetcode.cn/problems/3sum":                                   "3sum",
		"www.leetcode.cn/problems/3sum/":                              "3sum",
		"HTTPS://LeetCode.CN/problems/Two-Sum/":                       "two-sum",
		// NeetCode slugs map to the LeetCode slug they mirror.
		"https://neetcode.io/problems/two-integer-sum/question":                   "two-sum",
		"https://neetcode.io/problems/two-integer-sum":                            "two-sum",
		"https://www.neetcode.io/problems/anagram-groups/solution":                "group-anagrams",
		"https://neetcode.io/problems/validate-parentheses?list=blind75":          "valid-parentheses",
		"https://neetcode.io/problems/valid-sudoku/question?list=neetcode150#top": "valid-sudoku",
		"neetcode.io/problems/top-k-elements-in-list":                             "top-k-frequent-elements",
		"www.neetcode.io/problems/two-integer-sum/":                               "two-sum",
		"https://NeetCode.io/problems/Two-Integer-Sum/":                           "two-sum",
		// A NeetCode slug that is not in the table is never guessed.
		"https://neetcode.io/problems/no-such-neetcode-problem": "",
		"neetcode.io/problems/two-sum-nope":                     "",
		"https://neetcode.io/problems/":                         "",
		"https://neetcode.io/courses/dsa-for-beginners":         "",
		// A bare slug is a LeetCode slug, never a NeetCode one.
		"two-integer-sum":                       "two-integer-sum",
		"":                                      "",
		"https://example.com/problems/two-sum/": "",
		"https://neetcode.io.evil.com/problems/two-integer-sum": "",
		"https://notneetcode.io/problems/two-integer-sum":       "",
		"https://leetcode.com/contest/weekly-1/":                "",
		"two sum":                                               "",
		"two--sum":                                              "",
		strings.Repeat("a", MaxSlugLength+1):                    "",
	} {
		got, ok := NormalizeProblemRef(in)
		if got != want || ok != (want != "") {
			t.Errorf("%q: %q %v, want %q", in, got, ok, want)
		}
	}
}

func TestResolveProblemRefErrors(t *testing.T) {
	for in, want := range map[string]error{
		"https://neetcode.io/problems/no-such-neetcode-problem": ErrUnknownNeetCodeProblem,
		"neetcode.io/problems/nope":                             ErrUnknownNeetCodeProblem,
		"https://example.com/problems/two-sum":                  ErrInvalidProblemRef,
		"two sum":                                               ErrInvalidProblemRef,
		"":                                                      ErrInvalidProblemRef,
		"https://neetcode.io/courses/x":                         ErrInvalidProblemRef,
	} {
		if _, err := ResolveProblemRef(in); !errors.Is(err, want) {
			t.Errorf("%q: %v, want %v", in, err, want)
		}
	}
}

// neetcodeSlugsJS is the extension's table of NeetCode slugs.
const neetcodeSlugsJS = "../../extension/leetgrinder/neetcode-slugs.js"

// TestNeetCodeSlugsMatchExtension keeps the Go table in step with the
// extension's generated neetcode-slugs.js. Regenerate both with
// `node scripts/update-neetcode-slugs.js` in extension/leetgrinder.
func TestNeetCodeSlugsMatchExtension(t *testing.T) {
	src, err := os.ReadFile(neetcodeSlugsJS)
	if err != nil {
		t.Fatal(err)
	}
	row := regexp.MustCompile(`(?m)^\s+"([^"]+)": \["([^"]+)",`)
	js := map[string]string{}
	for _, m := range row.FindAllStringSubmatch(string(src), -1) {
		js[m[1]] = m[2]
	}
	if len(js) < 100 {
		t.Fatalf("parsed only %d rows from %s", len(js), neetcodeSlugsJS)
	}
	goTable := NeetCodeSlugs()
	for nc, lc := range js {
		if got, ok := goTable[nc]; !ok || got != lc {
			t.Errorf("%s: Go table has %q (present %v), JS has %q; regenerate neetcode_slugs.json", nc, got, ok, lc)
		}
	}
	for nc := range goTable {
		if _, ok := js[nc]; !ok {
			t.Errorf("%s is in the Go table but not in neetcode-slugs.js", nc)
		}
	}
	for nc, lc := range goTable {
		if !ValidSlug(nc) || !ValidSlug(lc) {
			t.Errorf("invalid slug pair %q -> %q", nc, lc)
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
