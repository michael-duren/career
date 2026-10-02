package leetgrinder

import (
	_ "embed"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxFetchAttempts is how many times the server asks LeetCode for a
// problem's metadata before giving up.
const MaxFetchAttempts = 4

const (
	MaxSlugLength  = 100
	MaxTitleLength = 200
	MaxTopics      = 20
	MaxTopicLength = 60
	MaxNumber      = 100000
)

var (
	slugPattern  = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	topicPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

// ValidSlug reports whether s is a LeetCode problem slug.
func ValidSlug(s string) bool { return len(s) <= MaxSlugLength && slugPattern.MatchString(s) }

//go:embed catalog_seed.json
var catalogSeedJSON []byte

// SeedProblem is one entry of catalog_seed.json: a problem from the retired
// curriculum with its curated optimal complexity. CurriculumTopic is the
// curriculum week it belonged to, kept for reference only; topic tags come
// from LeetCode.
type SeedProblem struct {
	Slug            string `json:"slug"`
	Number          int    `json:"number"`
	Title           string `json:"title"`
	Difficulty      string `json:"difficulty"`
	OptimalTime     string `json:"optimalTime"`
	OptimalSpace    string `json:"optimalSpace"`
	OptimalNote     string `json:"optimalNote"`
	CurriculumTopic string `json:"curriculumTopic"`
}

// CatalogSeedJSON is the embedded seed catalog, a JSON array of SeedProblem.
func CatalogSeedJSON() []byte { return catalogSeedJSON }

// CatalogSeed returns the embedded seed catalog.
func CatalogSeed() ([]SeedProblem, error) {
	var seed []SeedProblem
	if err := json.Unmarshal(catalogSeedJSON, &seed); err != nil {
		return nil, err
	}
	return seed, nil
}

// TopicTag is a LeetCode topic tag as the extension reports it.
type TopicTag struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// ProblemMetadata is what LeetCode says about a problem: its frontend id,
// title, difficulty, and topic tags.
type ProblemMetadata struct {
	Number     int        `json:"number"`
	Title      string     `json:"title"`
	Difficulty string     `json:"difficulty"`
	Topics     []TopicTag `json:"topics"`
}

var ErrInvalidMetadata = errors.New("invalid problem metadata")

// Normalize trims the metadata and checks it: a number from 0 (unknown) to
// 100000, a printable title of at most 200 characters, a LeetCode difficulty
// or "", and at most 20 topics with slug-shaped tags. Duplicate topics are
// dropped.
func (m *ProblemMetadata) Normalize() error {
	m.Title = strings.TrimSpace(m.Title)
	if m.Number < 0 || m.Number > MaxNumber || !printable(m.Title) || utf8.RuneCountInString(m.Title) > MaxTitleLength {
		return ErrInvalidMetadata
	}
	switch m.Difficulty {
	case "", "Easy", "Medium", "Hard":
	default:
		return ErrInvalidMetadata
	}
	if len(m.Topics) > MaxTopics {
		return ErrInvalidMetadata
	}
	seen := map[string]bool{}
	topics := []TopicTag{}
	for _, t := range m.Topics {
		t.Name = strings.TrimSpace(t.Name)
		if len(t.Slug) > MaxTopicLength || !topicPattern.MatchString(t.Slug) || !printable(t.Name) || utf8.RuneCountInString(t.Name) > MaxTopicLength {
			return ErrInvalidMetadata
		}
		if !seen[t.Slug] {
			seen[t.Slug] = true
			topics = append(topics, t)
		}
	}
	m.Topics = topics
	return nil
}

// TopicSlugs lists the topic tag slugs in order.
func (m ProblemMetadata) TopicSlugs() []string {
	out := make([]string, 0, len(m.Topics))
	for _, t := range m.Topics {
		out = append(out, t.Slug)
	}
	return out
}

func printable(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// NormalizeProblemRef turns a LeetCode problem URL, a path such as
// /problems/two-sum/description/, or a bare slug into the problem's slug.
func NormalizeProblemRef(input string) (string, bool) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", false
	}
	if strings.Contains(s, "/") {
		if !strings.Contains(s, "://") && strings.HasPrefix(strings.ToLower(s), "leetcode.com/") {
			s = "https://" + s
		}
		u, err := url.Parse(s)
		if err != nil {
			return "", false
		}
		if u.Host != "" {
			host := strings.ToLower(u.Hostname())
			if host != "leetcode.com" && host != "www.leetcode.com" {
				return "", false
			}
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 || parts[0] != "problems" {
			return "", false
		}
		s = parts[1]
	}
	s = strings.ToLower(s)
	if !ValidSlug(s) {
		return "", false
	}
	return s, true
}

// TopicLabel is a readable name for a topic tag slug, built from the slug.
// Prefer Problem.TopicLabel, which uses the name LeetCode sent when known.
func TopicLabel(slug string) string {
	if slug == "" {
		return "Untagged"
	}
	words := strings.Split(slug, "-")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
