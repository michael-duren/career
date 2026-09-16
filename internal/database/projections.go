package database

import (
	"context"
	"database/sql"
	"regexp"
	"sort"
	"strings"
)

const ParserVersion = 1

type Task struct {
	Index          int
	Checked        bool
	Label, Section string
}
type Log struct {
	Index int
	Date  *string
	Text  string
}

var fenceRE = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})")
var headingRE = regexp.MustCompile(`^##\s+(.+?)\s*$`)
var taskRE = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+\[([ xX])\]\s+(.*)$`)

func Checklist(body string) []Task {
	out := []Task{}
	fence, section := "", ""
	for i, line := range strings.Split(body, "\n") {
		if m := fenceRE.FindStringSubmatch(line); m != nil {
			if fence == "" {
				fence = m[1]
			} else if m[1][0] == fence[0] && len(m[1]) >= len(fence) {
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		if m := headingRE.FindStringSubmatch(line); m != nil {
			section = strings.ToLower(m[1])
		}
		if m := taskRE.FindStringSubmatch(line); m != nil {
			out = append(out, Task{i, strings.EqualFold(m[1], "x"), strings.TrimSpace(m[2]), section})
		}
	}
	return out
}
func section(body, heading string) string {
	lines := strings.Split(body, "\n")
	start := -1
	for i, l := range lines {
		if start >= 0 && strings.HasPrefix(l, "## ") {
			return strings.TrimSpace(strings.Join(lines[start:i], "\n"))
		}
		if l == "## "+heading || strings.TrimRight(l, "\r \t") == "## "+heading {
			start = i + 1
			continue
		}
		if start >= 0 && strings.HasPrefix(l, "## ") {
			return strings.TrimSpace(strings.Join(lines[start:i], "\n"))
		}
	}
	if start >= 0 {
		return strings.TrimSpace(strings.Join(lines[start:], "\n"))
	}
	return ""
}
func bookLogs(body string) []Log {
	out := []Log{}
	in := false
	date := ""
	start := 0
	buf := []string{}
	bullet := regexp.MustCompile(`^\s*[-*]\s+`)
	flush := func() {
		if date != "" {
			text := []string{}
			for _, l := range buf {
				l = strings.TrimSpace(bullet.ReplaceAllString(l, ""))
				if l != "" {
					text = append(text, l)
				}
			}
			if len(text) > 0 {
				d := date
				out = append(out, Log{start, &d, strings.Join(text, " ")})
			}
		}
		buf = nil
	}
	for i, l := range strings.Split(body, "\n") {
		l = strings.TrimRight(l, " \t\r")
		if headingRE.MatchString(l) {
			flush()
			date = ""
			low := strings.ToLower(l)
			in = strings.Contains(low, "log") || strings.Contains(low, "notes") || strings.Contains(low, "journal")
			continue
		}
		if in && strings.HasPrefix(l, "### ") {
			flush()
			date = strings.TrimSpace(l[4:])
			start = i
			continue
		}
		if in && date != "" {
			buf = append(buf, l)
		}
	}
	flush()
	sort.SliceStable(out, func(i, j int) bool { return *out[i].Date > *out[j].Date })
	return out
}
func companyLogs(body string) []Log {
	out := []Log{}
	in := false
	for i, l := range strings.Split(body, "\n") {
		if strings.HasPrefix(l, "## ") {
			in = strings.TrimSpace(l) == "## Log"
			continue
		}
		if !in {
			continue
		}
		if strings.HasPrefix(l, "- ") {
			out = append(out, Log{Index: i, Text: strings.TrimSpace(l[2:])})
		} else if len(out) > 0 {
			out[len(out)-1].Text += "\n" + strings.TrimPrefix(l, "  ")
		}
	}
	for i := range out {
		out[i].Text = strings.TrimSpace(out[i].Text)
	}
	return out
}
func project(ctx context.Context, tx *sql.Tx, kind, id, body, rev string) error {
	for _, suffix := range []string{"_checklist", "_logs"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+kind+suffix+" WHERE owner_slug=$1", id); err != nil {
			return err
		}
	}
	tasks := Checklist(body)
	if kind == "company" {
		research, relationship := -1, -1
		pattern := regexp.MustCompile(`(?i)^(Research team (&|and) open roles|Research the role)$`)
		for i, t := range tasks {
			if t.Section != "steps" {
				continue
			}
			if research < 0 && pattern.MatchString(t.Label) {
				research = i
			}
			if relationship < 0 && t.Label == "Reach out and start building a relationship" {
				relationship = i
			}
		}
		if research >= 0 && relationship > research {
			task := tasks[research]
			copy(tasks[research:relationship], tasks[research+1:relationship+1])
			tasks[relationship] = task
		}
	}

	total, done := 0, 0
	for i, t := range tasks {
		if (kind == "book" && allowed(t.Section, "chapters|modules|sections")) || (kind == "company" && t.Section == "steps") {
			total++
			if t.Checked {
				done++
			}
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO "+kind+"_checklist(owner_slug,position,source_line,section,label,checked) VALUES($1,$2,$3,$4,$5,$6)", id, i, t.Index, t.Section, t.Label, t.Checked); err != nil {
			return err
		}
	}
	logs := bookLogs(body)
	if kind == "company" {
		logs = companyLogs(body)
	}
	for i, l := range logs {
		if _, err := tx.ExecContext(ctx, "INSERT INTO "+kind+"_logs(owner_slug,position,source_line,log_date,text) VALUES($1,$2,$3,$4,$5)", id, i, l.Index, l.Date, l.Text); err != nil {
			return err
		}
	}
	var err error
	if kind == "book" {
		excerpt := ""
		if len(logs) > 0 {
			excerpt = logs[0].Text
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO book_summaries(book_slug,chapter_count,completed_count,excerpt,source_revision,parser_version) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(book_slug) DO UPDATE SET chapter_count=$2,completed_count=$3,excerpt=$4,source_revision=$5,parser_version=$6", id, total, done, excerpt, rev, ParserVersion)
	} else {
		_, err = tx.ExecContext(ctx, "INSERT INTO company_summaries(company_slug,step_count,completed_count,why,source_revision,parser_version) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(company_slug) DO UPDATE SET step_count=$2,completed_count=$3,why=$4,source_revision=$5,parser_version=$6", id, total, done, section(body, "Why"), rev, ParserVersion)
	}
	return err
}
func (s *Store) Rebuild(ctx context.Context) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, kind := range []string{"book", "company"} { // Lock owners in stable order; saves update the owner before its projections.
		rows, err := tx.QueryContext(ctx, "SELECT slug,body,revision FROM "+models[kind].Table+" ORDER BY slug FOR UPDATE")
		if err != nil {
			return err
		}
		type item struct{ id, body, rev string }
		items := []item{}
		for rows.Next() {
			var x item
			if err := rows.Scan(&x.id, &x.body, &x.rev); err != nil {
				rows.Close()
				return err
			}
			items = append(items, x)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		for _, x := range items {
			if err := project(ctx, tx, kind, x.id, x.body, x.rev); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
func (s *Store) Toggle(ctx context.Context, kind, id string, revision string, index int, checked bool) (Result, error) {
	if kind != "book" && kind != "company" {
		return Result{}, ErrNotFound
	}
	r, err := s.Detail(ctx, kind, id)
	if err != nil {
		return r, err
	}
	if r.Revision != revision {
		return Result{}, ErrConflict
	}
	body := r.Entry["body"].(string)
	found := false
	for _, t := range Checklist(body) {
		if t.Index == index {
			found = true
		}
	}
	if !found {
		return Result{}, ErrConflict
	}
	lines := strings.Split(body, "\n")
	replacement := "[ ]"
	if checked {
		replacement = "[x]"
	}
	box := regexp.MustCompile(`\[[ xX]\]`)
	at := box.FindStringIndex(lines[index])
	lines[index] = lines[index][:at[0]] + replacement + lines[index][at[1]:]
	r.Entry["body"] = strings.Join(lines, "\n")
	return s.Save(ctx, kind, r.Entry, &revision)
}
