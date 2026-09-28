// Command verify-leetgrinder checks live reading links and LeetCode catalog metadata.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

type catalogRow struct {
	Stat struct {
		ID    int    `json:"frontend_question_id"`
		Slug  string `json:"question__title_slug"`
		Title string `json:"question__title"`
	} `json:"stat"`
	Difficulty struct {
		Level int `json:"level"`
	} `json:"difficulty"`
	Paid bool `json:"paid_only"`
}

func main() {
	if err := verify(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func verify() error {
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Get("https://leetcode.com/api/problems/all/")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("LeetCode catalog: HTTP %d", response.StatusCode)
	}
	var catalog struct {
		Rows []catalogRow `json:"stat_status_pairs"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 16<<20)).Decode(&catalog); err != nil {
		return err
	}
	bySlug := map[string]catalogRow{}
	for _, row := range catalog.Rows {
		bySlug[row.Stat.Slug] = row
	}
	seen := map[string]bool{}
	readings := map[string]bool{}
	difficulties := map[int]string{1: "Easy", 2: "Medium", 3: "Hard"}
	core, optional, days := 0, 0, 0
	weeks := leetgrinder.Curriculum()
	for _, week := range weeks {
		for _, day := range week.Days {
			days++
			core += len(day.Core)
			optional += len(day.Optional)
			for _, group := range [][]leetgrinder.Problem{day.Core, day.Optional} {
				for _, problem := range group {
					row, ok := bySlug[problem.Slug]
					if !ok || row.Stat.ID != problem.ID || row.Stat.Title != problem.Title || difficulties[row.Difficulty.Level] != problem.Difficulty || row.Paid || seen[problem.Slug] {
						return fmt.Errorf("day %d: catalog mismatch, paid or duplicate problem: %+v", day.Number, problem)
					}
					seen[problem.Slug] = true
				}
			}
			for _, r := range day.Readings {
				readings[strings.SplitN(r.URL, "#", 2)[0]] = true
			}
		}
	}
	if len(weeks) != 12 || days != 84 || core != 252 || optional != 48 || len(seen) != 300 {
		return fmt.Errorf("incorrect curriculum counts: %d weeks, %d days, %d core, %d optional, %d unique", len(weeks), days, core, optional, len(seen))
	}
	fmt.Printf("PASS: 12 weeks, 84 sessions, 252 core + 48 optional; all 300 unique LeetCode problems match live ID, title, slug, difficulty and free-access metadata.\n")
	urls := []string{}
	for u := range readings {
		urls = append(urls, u)
	}
	sort.Strings(urls)
	failures := 0
	for _, u := range urls {
		request, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			fmt.Printf("FAIL %s: %v\n", u, err)
			failures++
			continue
		}
		request.Header.Set("User-Agent", "leetgrinder-link-verifier/1.0 (https://github.com/michael-duren/career)")
		response, err := client.Do(request)
		if err != nil {
			fmt.Printf("FAIL %s: %v\n", u, err)
			failures++
			continue
		}
		_, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, 32<<20))
		response.Body.Close()
		if response.StatusCode != 200 || readErr != nil {
			fmt.Printf("FAIL %s: HTTP %d %v\n", u, response.StatusCode, readErr)
			failures++
			continue
		}
		fmt.Printf("PASS %s\n", u)
	}
	if failures > 0 {
		return fmt.Errorf("%d reading links could not be verified; network errors are not proof of a broken link", failures)
	}
	fmt.Printf("PASS: %d distinct reading documents reachable.\n", len(urls))
	return nil
}
