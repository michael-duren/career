package database

import (
	"encoding/json"
	"fmt"
	"math"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"
)

type Entity = map[string]any
type field struct {
	Name, Column, Type string
	Optional           bool
	Max                int
	Values             string
}
type model struct {
	Table, Key, Collection string
	Fields                 []field
}

var models = map[string]model{}

func fields(spec string) []field {
	var out []field
	for _, s := range strings.Fields(spec) {
		p := strings.Split(s, ":")
		f := field{Name: p[0], Column: p[0], Type: p[1]}
		if strings.HasSuffix(f.Name, "?") {
			f.Optional = true
			f.Name = strings.TrimSuffix(f.Name, "?")
			f.Column = f.Name
		}
		if len(p) > 2 {
			f.Column = p[2]
		}
		out = append(out, f)
	}
	return out
}
func init() {
	base := "title:string description:string tags:array body:string updatedAt?:timestamp:updated_at"
	cat := "title:string category:string type:string url?:url cover?:url status:string featured:bool priority:string tags:array body:string updatedAt?:timestamp:updated_at"
	models["run"] = model{"running_notes", "id", "runningNotes", fields("id:id title:string runDate:date:run_date startedAt:timestamp:started_at distanceKm?:number:distance_km durationMin?:number:duration_min tags:array body:string updatedAt?:timestamp:updated_at")}
	models["note"] = model{"notes", "id", "notes", fields("id:id topic:string " + base)}
	models["document"] = model{"documents", "id", "documents", fields("id:id " + base)}
	models["personal"] = model{"personal_journal_entries", "id", "personalJournal", fields("id:id date:undated:entry_date " + base)}
	models["week"] = model{"journal_weeks", "slug", "weeks", fields("slug:id week:int year:int dates:range tags:array body:string updatedAt?:timestamp:updated_at")}
	models["book"] = model{"books", "slug", "books", fields("slug:id " + cat + " edition?:string authors:array isbn?:string started?:date finished?:date rating?:number")}
	models["company"] = model{"companies", "slug", "companies", fields("slug:id " + strings.Replace(cat, "url?:url", "url:url", 1))}
	models["connection"] = model{"connections", "id", "connections", fields("id:uuid name:string role:string companyName:string:company_name companySlug?:id:company_slug email:string url?:url connectedOn?:date:connected_on lastContactedOn?:date:last_contacted_on cadenceDays?:int:cadence_days queued:bool notes:string tags:array updatedAt?:timestamp:updated_at")}
	models["goal"] = model{"goals", "id", "goals", fields("id:uuid status:string title:string startDate:date:start_date endDate:date:end_date color:color dailyHours?:number:daily_hours createdAt:timestamp:created_at updatedAt:timestamp:updated_at")}
}

var idRE = regexp.MustCompile(`^[a-zA-Z0-9_/-]{1,200}$`)
var uuidRE = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var colorRE = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func size(s string) int { return len(utf16.Encode([]rune(s))) }
func validDate(s string) bool {
	t, e := time.Parse("2006-01-02", s)
	return e == nil && t.Format("2006-01-02") == s
}
func number(v any) (float64, bool) {
	n, ok := v.(float64)
	return n, ok && !math.IsNaN(n) && !math.IsInf(n, 0)
}
func allowed(s, values string) bool {
	for _, v := range strings.Split(values, "|") {
		if s == v {
			return true
		}
	}
	return false
}
func validURL(s string) bool {
	u, e := url.Parse(s)
	return e == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
func Validate(kind string, e Entity) error {
	m, ok := models[kind]
	if !ok {
		return fmt.Errorf("unknown entry kind")
	}
	known := map[string]bool{}
	for _, f := range m.Fields {
		known[f.Name] = true
		v, exists := e[f.Name]
		if !exists && f.Optional {
			continue
		}
		if !exists || v == nil {
			return fmt.Errorf("%s is required (null is unsupported)", f.Name)
		}
		str, isstr := v.(string)
		switch f.Type {
		case "bool":
			if _, ok := v.(bool); !ok {
				return fmt.Errorf("%s must be boolean", f.Name)
			}
		case "int", "number":
			n, ok := number(v)
			if !ok || n < 0 || (f.Type == "int" && n != math.Trunc(n)) {
				return fmt.Errorf("invalid %s", f.Name)
			}
			max := 168.0
			if f.Name == "distanceKm" {
				max = 500
			}
			if f.Name == "durationMin" {
				max = 1440
			}
			if f.Name == "week" {
				max = 10000
				if n < 1 {
					return fmt.Errorf("week must be positive")
				}
			}
			if f.Name == "year" {
				max = 2200
				if n < 2000 {
					return fmt.Errorf("year before 2000")
				}
			}
			if f.Name == "rating" {
				max = 5
			}
			if f.Name == "dailyHours" {
				max = 24
			}
			if f.Name == "cadenceDays" {
				max = 3650
				if n < 1 {
					return fmt.Errorf("cadenceDays must be positive")
				}
			}
			if n > max {
				return fmt.Errorf("%s exceeds %v", f.Name, max)
			}
		case "array":
			a, ok := v.([]any)
			if !ok || len(a) > 30 {
				return fmt.Errorf("invalid %s", f.Name)
			}
			for _, v := range a {
				s, ok := v.(string)
				max := 80
				if f.Name == "authors" {
					max = 200
				}
				if !ok || size(s) > max || (f.Name == "tags" && strings.TrimSpace(s) == "") {
					return fmt.Errorf("invalid %s item", f.Name)
				}
			}
		default:
			if !isstr {
				return fmt.Errorf("%s must be a string", f.Name)
			}
			switch f.Type {
			case "id":
				if !idRE.MatchString(str) {
					return fmt.Errorf("invalid ID")
				}
			case "uuid":
				if !uuidRE.MatchString(str) {
					return fmt.Errorf("invalid UUID")
				}
			case "color":
				if !colorRE.MatchString(str) {
					return fmt.Errorf("invalid color")
				}
			case "date", "undated":
				if !(f.Type == "undated" && str == "") && !validDate(str) {
					return fmt.Errorf("invalid %s", f.Name)
				}
			case "timestamp":
				if _, err := time.Parse(time.RFC3339Nano, str); err != nil {
					return fmt.Errorf("invalid timestamp %s", f.Name)
				}
			case "url":
				if !validURL(str) {
					return fmt.Errorf("invalid %s URL", f.Name)
				}
			case "range":
				p := strings.Split(str, " to ")
				if len(p) != 2 || !validDate(p[0]) || !validDate(p[1]) || p[1] < p[0] {
					return fmt.Errorf("invalid week date range")
				}
			case "string":
				max := 1000
				switch f.Name {
				case "title", "category", "name", "role", "companyName":
					max = 200
				case "email":
					max = 254
				case "notes":
					max = 5000
				case "topic":
					max = 100
				case "body":
					max = 100000
				}
				if size(str) > max {
					return fmt.Errorf("%s too long", f.Name)
				}
				if allowed(f.Name, "title|topic|category|name") && strings.TrimSpace(str) == "" {
					return fmt.Errorf("%s is empty", f.Name)
				}
			}
		}
	}
	if kind == "book" || kind == "company" {
		if !allowed(e["priority"].(string), "high|medium|low") {
			return fmt.Errorf("invalid priority")
		}
		if kind == "book" {
			if !allowed(e["category"].(string), "Computer Science|Networking|OS|Systems|Distributed Systems|Languages|Career|Online Course") || !allowed(e["status"].(string), "backlog|reading|paused|completed|reference") || !allowed(e["type"].(string), "book|course") {
				return fmt.Errorf("invalid book enum")
			}
			known["progress"] = true
			if p, exists := e["progress"]; exists {
				v, ok := p.(map[string]any)
				if !ok || len(v) != 3 {
					return fmt.Errorf("invalid progress")
				}
				u, _ := v["unit"].(string)
				t, tok := number(v["total"])
				c, cok := number(v["completed"])
				if !allowed(u, "chapter|page|module|section|lecture") || !tok || !cok || t < 0 || t > 100000 || c < 0 || c > t || t != math.Trunc(t) || c != math.Trunc(c) {
					return fmt.Errorf("invalid progress")
				}
			}
		} else {
			if e["type"] != "company" || !allowed(e["status"].(string), "not_started|applied|interviewing|offer|rejected|passed") {
				return fmt.Errorf("invalid company enum")
			}
			// Legacy archives embed contacts; Import converts them to connections.
			known["contacts"] = true
			if v, exists := e["contacts"]; exists {
				if err := validateChildren("contacts", v, 200); err != nil {
					return err
				}
			}
		}
	}
	if kind == "connection" {
		// photo is a read-only projection of connection_photos.
		known["photo"] = true
		if s := e["email"].(string); s != "" {
			if a, err := mail.ParseAddress(s); err != nil || a.Address != s {
				return fmt.Errorf("invalid connection email")
			}
		}
	}
	if kind == "week" {
		for _, key := range []string{"hours", "targets"} {
			known[key] = true
			v, exists := e[key]
			if !exists && key == "targets" {
				continue
			}
			obj, ok := v.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid %s", key)
			}
			for k, v := range obj {
				n, ok := number(v)
				if size(k) > 80 || !ok || n < 0 || n > 168 {
					return fmt.Errorf("invalid %s metric", key)
				}
			}
		}
	}
	if kind == "note" {
		// Older clients and archives have no todos; absence means none.
		known["todos"] = true
		if v, exists := e["todos"]; exists {
			if err := validateChildren("todos", v, 200); err != nil {
				return err
			}
		}
	}
	if kind == "goal" {
		if !allowed(e["status"].(string), "planned|active|done|dropped") {
			return fmt.Errorf("invalid goal status")
		}
		known["dependsOn"] = true
		deps, ok := e["dependsOn"].([]any)
		if !ok {
			return fmt.Errorf("dependsOn must be an array")
		}
		seen := map[string]bool{}
		for _, v := range deps {
			id, ok := v.(string)
			if !ok || !uuidRE.MatchString(id) || seen[id] {
				return fmt.Errorf("invalid or duplicate prerequisite")
			}
			seen[id] = true
		}
		if e["startDate"].(string) < "1900-01-01" || e["endDate"].(string) > "2200-12-31" || e["endDate"].(string) < e["startDate"].(string) {
			return fmt.Errorf("invalid goal range")
		}
		for key, max := range map[string]int{"notes": 500, "steps": 200} {
			known[key] = true
			if err := validateChildren(key, e[key], max); err != nil {
				return err
			}
		}
		known["metadata"] = true
		obj, ok := e["metadata"].(map[string]any)
		if !ok || len(obj) > 50 {
			return fmt.Errorf("invalid metadata")
		}
		for k, v := range obj {
			s, ok := v.(string)
			if !ok || strings.TrimSpace(k) == "" || size(k) > 80 || size(s) > 2000 {
				return fmt.Errorf("invalid metadata item")
			}
		}
	}
	for k := range e {
		if !known[k] {
			return fmt.Errorf("unsupported %s field %s", kind, k)
		}
	}
	return nil
}
func validateChildren(kind string, v any, max int) error {
	a, ok := v.([]any)
	if !ok || len(a) > max {
		return fmt.Errorf("invalid %s", kind)
	}
	seen := map[string]bool{}
	for _, raw := range a {
		o, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid child")
		}
		id, _ := o["id"].(string)
		if !uuidRE.MatchString(id) || seen[id] {
			return fmt.Errorf("invalid/duplicate child UUID")
		}
		seen[id] = true
		spec := map[string]int{"id": 36}
		switch kind {
		case "contacts":
			spec["name"] = 200
			spec["role"] = 200
			spec["email"] = 254
			spec["url"] = 100000
			spec["notes"] = 5000
		case "notes":
			spec["body"] = 20000
			spec["createdAt"] = 100
		case "steps", "todos":
			spec["title"] = 500
		}
		hasDone := allowed(kind, "steps|todos")
		if len(o) != len(spec)+map[bool]int{true: 1, false: 0}[hasDone] {
			return fmt.Errorf("unsupported child fields")
		}
		for k, max := range spec {
			s, ok := o[k].(string)
			if !ok || size(s) > max {
				return fmt.Errorf("invalid child %s", k)
			}
			if allowed(k, "name|title|body") && strings.TrimSpace(s) == "" {
				return fmt.Errorf("empty child %s", k)
			}
			if k == "createdAt" {
				if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
					return err
				}
			}
			if k == "url" && s != "" && !validURL(s) {
				return fmt.Errorf("invalid contact URL")
			}
			if k == "email" && s != "" {
				a, err := mail.ParseAddress(s)
				if err != nil || a.Address != s {
					return fmt.Errorf("invalid contact email")
				}
			}
		}
		if hasDone {
			if _, ok := o["done"].(bool); !ok {
				return fmt.Errorf("invalid %s done", kind)
			}
		}
	}
	return nil
}
func Decode(raw []byte) (Entity, error) {
	var e Entity
	err := json.Unmarshal(raw, &e)
	if e == nil && err == nil {
		err = fmt.Errorf("object required")
	}
	return e, err
}

// Normalize interactive saves like the existing Zod schemas. Imports deliberately
// bypass this function so a dry run never silently changes source data.
func NormalizeForSave(kind string, input Entity) Entity {
	e := Entity{}
	for k, v := range input {
		e[k] = v
	}
	for _, key := range []string{"title", "topic"} {
		if s, ok := e[key].(string); ok {
			e[key] = strings.TrimSpace(s)
		}
	}
	if kind == "company" {
		if s, ok := e["category"].(string); ok {
			e["category"] = strings.TrimSpace(s)
		}
	}
	if tags, ok := e["tags"].([]any); ok {
		out := make([]any, len(tags))
		for i, v := range tags {
			if s, ok := v.(string); ok {
				out[i] = strings.TrimSpace(s)
			} else {
				out[i] = v
			}
		}
		e["tags"] = out
	}
	if kind == "book" {
		if authors, ok := e["authors"].([]any); ok {
			out := []any{}
			for _, v := range authors {
				if s, ok := v.(string); ok {
					if s = strings.TrimSpace(s); s != "" {
						out = append(out, s)
					}
				} else {
					out = append(out, v)
				}
			}
			e["authors"] = out
		}
		for _, key := range []string{"url", "cover", "started", "finished"} {
			if e[key] == "" {
				delete(e, key)
			}
		}
	}
	if kind == "company" && e["cover"] == "" {
		delete(e, "cover")
	}
	if kind == "connection" {
		for _, key := range []string{"name", "role", "companyName", "email"} {
			if s, ok := e[key].(string); ok {
				e[key] = strings.TrimSpace(s)
			}
		}
		for _, key := range []string{"url", "companySlug", "connectedOn", "lastContactedOn"} {
			if e[key] == "" {
				delete(e, key)
			}
		}
		delete(e, "photo")
	}
	children := map[string][]string{}
	if kind == "goal" {
		children["notes"] = []string{"body"}
		children["steps"] = []string{"title"}
	}
	if kind == "note" {
		children["todos"] = []string{"title"}
	}
	for key, trim := range children {
		if items, ok := e[key].([]any); ok {
			out := make([]any, len(items))
			for i, v := range items {
				if o, ok := v.(map[string]any); ok {
					copy := Entity{}
					for k, v := range o {
						copy[k] = v
					}
					for _, k := range trim {
						if s, ok := copy[k].(string); ok {
							copy[k] = strings.TrimSpace(s)
						}
					}
					out[i] = copy
				} else {
					out[i] = v
				}
			}
			e[key] = out
		}
	}
	if kind == "goal" {
		if metadata, ok := e["metadata"].(map[string]any); ok {
			out := Entity{}
			for k, v := range metadata {
				out[strings.TrimSpace(k)] = v
			}
			e["metadata"] = out
		}
	}
	return e
}

func ValidID(kind, id string) bool {
	if _, ok := models[kind]; !ok {
		return false
	}
	if kind == "goal" || kind == "connection" {
		return uuidRE.MatchString(id)
	}
	return idRE.MatchString(id)
}

func PrepareSave(kind string, input Entity) (Entity, error) {
	if _, exists := input["contacts"]; exists && kind == "company" {
		return nil, fmt.Errorf("contacts moved to connections; reload before saving")
	}
	if kind == "book" {
		if authors, ok := input["authors"].([]any); ok {
			if len(authors) > 30 {
				return nil, fmt.Errorf("too many authors")
			}
			for _, v := range authors {
				if s, ok := v.(string); ok && size(s) > 200 {
					return nil, fmt.Errorf("author too long")
				}
			}
		}
	}
	e := NormalizeForSave(kind, input)
	goalDefaults(kind, e)
	return e, Validate(kind, e)
}

// Older archives and clients have no dependency fields.
func goalDefaults(kind string, e Entity) {
	if kind != "goal" || e == nil {
		return
	}
	if _, ok := e["status"]; !ok {
		e["status"] = "planned"
	}
	if _, ok := e["dependsOn"]; !ok {
		e["dependsOn"] = []any{}
	}
}
