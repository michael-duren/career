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

//go:embed neetcode_slugs.json
var neetcodeSlugsJSON []byte

var neetcodeSlugs = func() map[string]string {
	var m map[string]string
	if err := json.Unmarshal(neetcodeSlugsJSON, &m); err != nil {
		panic("leetgrinder: neetcode_slugs.json: " + err.Error())
	}
	return m
}()

// NeetCodeSlugs returns a copy of the NeetCode slug to LeetCode slug table.
// It is generated with the extension's neetcode-slugs.js.
func NeetCodeSlugs() map[string]string {
	out := make(map[string]string, len(neetcodeSlugs))
	for k, v := range neetcodeSlugs {
		out[k] = v
	}
	return out
}

// ErrUnknownNeetCodeProblem is returned for a NeetCode link whose slug is not
// in the NeetCode to LeetCode table.
var ErrUnknownNeetCodeProblem = errors.New("unknown NeetCode problem")

// ErrInvalidProblemRef is returned for input that is not a problem link or slug.
var ErrInvalidProblemRef = errors.New("invalid problem link or slug")

// UnknownNeetCodeMessage explains a refused NeetCode link.
const UnknownNeetCodeMessage = "That NeetCode problem has no known LeetCode match. Use its LeetCode link or slug instead."

// MaxEchoedRefRunes is how much of a rejected reference an error repeats.
const MaxEchoedRefRunes = 60

// EchoRef shortens a rejected reference for an error message, cutting on a
// rune boundary.
func EchoRef(ref string) string {
	if r := []rune(ref); len(r) > MaxEchoedRefRunes {
		return string(r[:MaxEchoedRefRunes]) + "..."
	}
	return ref
}

// ResolveProblemRef turns a problem link or slug into the LeetCode slug. It
// accepts leetcode.com and leetcode.cn problem URLs, neetcode.io problem URLs
// (mapped to the LeetCode slug they mirror), paths such as
// /problems/two-sum/description/, and bare LeetCode slugs. A NeetCode slug
// that is not in the table yields ErrUnknownNeetCodeProblem rather than a
// guess; any other bad input yields ErrInvalidProblemRef.
func ResolveProblemRef(input string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", ErrInvalidProblemRef
	}
	neetcode := false
	if strings.Contains(s, "/") {
		if !strings.Contains(s, "://") && refHostPrefix(s) {
			s = "https://" + s
		}
		u, err := url.Parse(s)
		if err != nil {
			return "", ErrInvalidProblemRef
		}
		switch strings.ToLower(u.Scheme) {
		case "", "http", "https":
		default:
			return "", ErrInvalidProblemRef
		}
		if u.Host != "" {
			switch strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.") {
			case "leetcode.com", "leetcode.cn":
			case "neetcode.io":
				neetcode = true
			default:
				return "", ErrInvalidProblemRef
			}
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 || parts[0] != "problems" {
			return "", ErrInvalidProblemRef
		}
		s = parts[1]
	}
	s = strings.ToLower(s)
	if !ValidSlug(s) {
		return "", ErrInvalidProblemRef
	}
	if neetcode {
		lc, ok := neetcodeSlugs[s]
		if !ok {
			return "", ErrUnknownNeetCodeProblem
		}
		return lc, nil
	}
	return s, nil
}

// refHostPrefix reports whether a scheme-less reference starts with a
// supported problem site host, such as "leetcode.cn/problems/two-sum".
func refHostPrefix(s string) bool {
	s = strings.ToLower(s)
	for _, host := range []string{"leetcode.com/", "leetcode.cn/", "neetcode.io/"} {
		if strings.HasPrefix(s, host) || strings.HasPrefix(s, "www."+host) {
			return true
		}
	}
	return false
}

// NormalizeProblemRef is ResolveProblemRef without the reason for a refusal.
func NormalizeProblemRef(input string) (string, bool) {
	slug, err := ResolveProblemRef(input)
	return slug, err == nil
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
