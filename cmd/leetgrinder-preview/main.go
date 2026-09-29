// Command leetgrinder-preview serves the real session template on localhost for lesson review.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func previewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /leetgrinder/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		fmt.Fprint(w, leetgrinder.Styles)
	})
	mux.HandleFunc("GET /leetgrinder/lesson-player.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		fmt.Fprint(w, leetgrinder.LessonPlayerJS)
	})
	mux.HandleFunc("GET /day/{day}", func(w http.ResponseWriter, r *http.Request) {
		n, err := strconv.Atoi(r.PathValue("day"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		day, ok := leetgrinder.FindDay(n)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := leetgrinder.Session(leetgrinder.DayPage{Day: day, IDs: map[string]string{}}).Render(r.Context(), w); err != nil {
			log.Printf("render day %d: %v", n, err)
		}
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<!doctype html><title>Lesson preview</title><h1>Lesson preview</h1><ol>")
		for _, week := range leetgrinder.Curriculum() {
			for _, day := range week.Days {
				fmt.Fprintf(w, "<li><a href=\"/day/%d\">Day %d: %s</a></li>", day.Number, day.Number, strings.ReplaceAll(day.Title, "&", "&amp;"))
			}
		}
		fmt.Fprint(w, "</ol>")
	})
	return mux
}
func main() {
	addr := flag.String("addr", "127.0.0.1:8097", "localhost address")
	flag.Parse()
	log.Printf("Leetgrinder preview at http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, previewHandler()))
}
