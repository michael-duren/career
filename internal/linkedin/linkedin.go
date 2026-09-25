// Package linkedin parses the CSV files in a LinkedIn data export.
package linkedin

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type Connection struct {
	Name, Role, Company, Email, URL string
	// ConnectedOn is YYYY-MM-DD or empty.
	ConnectedOn string
}

var ErrNoHeader = errors.New("missing LinkedIn header row")

// header skips LinkedIn's "Notes:" preamble and returns column indexes for the first row naming every column in want.
func header(r *csv.Reader, want ...string) (map[string]int, error) {
	for {
		row, err := r.Read()
		if err == io.EOF {
			return nil, ErrNoHeader
		}
		if err != nil {
			return nil, err
		}
		cols := map[string]int{}
		for i, name := range row {
			cols[strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(name, "\ufeff")))] = i
		}
		found := true
		for _, name := range want {
			if _, ok := cols[name]; !ok {
				found = false
			}
		}
		if found {
			return cols, nil
		}
	}
}

func reader(r io.Reader) *csv.Reader {
	c := csv.NewReader(r)
	c.FieldsPerRecord = -1
	c.LazyQuotes = true
	c.ReuseRecord = true
	return c
}

// LazyQuotes tolerates LinkedIn's stray quotes, but an unclosed quote swallows
// the rest of the file into one field. Oversized fields are reported instead.
const (
	maxConnectionField = 10 << 10
	maxMessageField    = 256 << 10
)

func checkRow(c *csv.Reader, row []string, max int) error {
	for i, field := range row {
		if len(field) > max {
			line, _ := c.FieldPos(i)
			return fmt.Errorf("line %d: field longer than %d bytes; check for an unbalanced quote", line, max)
		}
	}
	return nil
}

func cell(row []string, cols map[string]int, name string) string {
	i, ok := cols[name]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

// ParseConnections reads Connections.csv. Rows without a name are skipped.
func ParseConnections(r io.Reader) ([]Connection, error) {
	c := reader(r)
	cols, err := header(c, "FIRST NAME", "LAST NAME", "URL")
	if err != nil {
		return nil, err
	}
	var out []Connection
	for {
		row, err := c.Read()
		if err == io.EOF {
			return out, nil
		}
		if err == nil {
			err = checkRow(c, row, maxConnectionField)
		}
		if err != nil {
			return nil, err
		}
		name := strings.TrimSpace(cell(row, cols, "FIRST NAME") + " " + cell(row, cols, "LAST NAME"))
		if name == "" {
			continue
		}
		conn := Connection{Name: name, Role: cell(row, cols, "POSITION"), Company: cell(row, cols, "COMPANY"), Email: cell(row, cols, "EMAIL ADDRESS"), URL: CanonicalURL(cell(row, cols, "URL"))}
		if on := cell(row, cols, "CONNECTED ON"); on != "" {
			for _, layout := range []string{"02 Jan 2006", "2 Jan 2006", "1/2/06", "2006-01-02"} {
				if t, err := time.Parse(layout, on); err == nil {
					conn.ConnectedOn = t.Format(time.DateOnly)
					break
				}
			}
		}
		out = append(out, conn)
	}
}

// ParseLastMessages reads messages.csv and returns the latest message date
// (YYYY-MM-DD) for every profile that sent or received a message.
func ParseLastMessages(r io.Reader) (map[string]string, error) {
	c := reader(r)
	cols, err := header(c, "DATE", "SENDER PROFILE URL")
	if err != nil {
		return nil, err
	}
	last := map[string]string{}
	for {
		row, err := c.Read()
		if err == io.EOF {
			return last, nil
		}
		if err == nil {
			err = checkRow(c, row, maxMessageField)
		}
		if err != nil {
			return nil, err
		}
		raw := cell(row, cols, "DATE")
		var day string
		for _, layout := range []string{"2006-01-02 15:04:05 MST", "2006-01-02 15:04:05", time.RFC3339} {
			if t, err := time.Parse(layout, raw); err == nil {
				day = t.UTC().Format(time.DateOnly)
				break
			}
		}
		if day == "" {
			continue
		}
		profiles := append([]string{cell(row, cols, "SENDER PROFILE URL")}, strings.FieldsFunc(cell(row, cols, "RECIPIENT PROFILE URLS"), func(r rune) bool { return r == ',' || r == ' ' || r == ';' })...)
		for _, profile := range profiles {
			if key := CanonicalURL(profile); key != "" && day > last[key] {
				last[key] = day
			}
		}
	}
}

// CanonicalURL maps LinkedIn profile URL variants to https://www.linkedin.com/in/<id>.
// Other http(s) URLs are returned unchanged; anything else becomes empty.
func CanonicalURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	host := strings.ToLower(strings.TrimPrefix(u.Host, "www."))
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if (host == "linkedin.com" || strings.HasSuffix(host, ".linkedin.com")) && len(parts) >= 2 && parts[0] == "in" && parts[1] != "" {
		id, err := url.PathUnescape(parts[1])
		if err != nil {
			id = parts[1]
		}
		return "https://www.linkedin.com/in/" + url.PathEscape(strings.ToLower(id))
	}
	return u.String()
}

var nonWord = regexp.MustCompile(`[^a-z0-9]+`)
var suffixes = map[string]bool{"inc": true, "llc": true, "ltd": true, "limited": true, "corp": true, "corporation": true, "co": true, "company": true, "gmbh": true, "plc": true, "sa": true, "ag": true, "bv": true, "the": true}

// NormalizeCompany lowercases, strips punctuation, and drops legal suffixes.
func NormalizeCompany(name string) string {
	words := strings.Fields(nonWord.ReplaceAllString(strings.ToLower(strings.ReplaceAll(name, "&", " and ")), " "))
	out := words[:0]
	for _, w := range words {
		if !suffixes[w] {
			out = append(out, w)
		}
	}
	return strings.Join(out, " ")
}

type Company struct{ Slug, Title string }

// Matcher links LinkedIn company names to tracked companies.
type Matcher struct{ names []struct{ key, slug string } }

func NewMatcher(companies []Company) *Matcher {
	m := &Matcher{}
	for _, c := range companies {
		for _, key := range []string{NormalizeCompany(c.Title), NormalizeCompany(strings.ReplaceAll(c.Slug[strings.LastIndex(c.Slug, "/")+1:], "-", " "))} {
			if key != "" {
				m.names = append(m.names, struct{ key, slug string }{key, c.Slug})
			}
		}
	}
	return m
}

// Match returns the slug whose normalized title equals the name or is its
// longest leading word sequence ("Amazon Web Services" matches "Amazon").
func (m *Matcher) Match(company string) string {
	name := NormalizeCompany(company)
	if name == "" {
		return ""
	}
	best, bestLen := "", 0
	for _, n := range m.names {
		if (name == n.key || strings.HasPrefix(name, n.key+" ")) && len(n.key) > bestLen {
			best, bestLen = n.slug, len(n.key)
		}
	}
	return best
}
