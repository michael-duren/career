package linkedin

import (
	"reflect"
	"strings"
	"testing"
)

const connectionsCSV = "\ufeffNotes:\n" +
	`"When exporting your connection data, you may notice that some of the email addresses are missing."` + "\n\n" +
	"First Name,Last Name,URL,Email Address,Company,Position,Connected On\n" +
	"Ada,Lovelace,https://www.linkedin.com/in/Ada-L/,ada@example.com,\"Grafana Labs, Inc.\",Staff SRE,19 Sep 2026\n" +
	",,,,,,\n" +
	"Grace,Hopper,https://linkedin.com/in/grace?trk=x,,Amazon Web Services (AWS),Engineer,\n"

func TestParseConnections(t *testing.T) {
	got, err := ParseConnections(strings.NewReader(connectionsCSV))
	if err != nil {
		t.Fatal(err)
	}
	want := []Connection{
		{Name: "Ada Lovelace", Role: "Staff SRE", Company: "Grafana Labs, Inc.", Email: "ada@example.com", URL: "https://www.linkedin.com/in/ada-l", ConnectedOn: "2026-09-19"},
		{Name: "Grace Hopper", Role: "Engineer", Company: "Amazon Web Services (AWS)", URL: "https://www.linkedin.com/in/grace"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
	if _, err := ParseConnections(strings.NewReader("a,b\n1,2\n")); err != ErrNoHeader {
		t.Fatalf("want ErrNoHeader, got %v", err)
	}
	unbalanced := "First Name,Last Name,URL,Email Address,Company,Position,Connected On\n" +
		"Ada,L,https://www.linkedin.com/in/ada,,Acme,\"CEO,01 Jan 2026\n" + strings.Repeat("Grace,H,https://www.linkedin.com/in/grace,,Acme,Eng,01 Jan 2026\n", 200)
	if _, err := ParseConnections(strings.NewReader(unbalanced)); err == nil || !strings.Contains(err.Error(), "unbalanced quote") {
		t.Fatalf("unbalanced quote: %v", err)
	}
}

func TestParseLastMessages(t *testing.T) {
	csv := "CONVERSATION ID,CONVERSATION TITLE,FROM,SENDER PROFILE URL,TO,RECIPIENT PROFILE URLS,DATE,SUBJECT,CONTENT,FOLDER\n" +
		"1,,Ada,https://www.linkedin.com/in/ada-l,Me,https://www.linkedin.com/in/me,2026-03-01 10:00:00 UTC,,hi,INBOX\n" +
		"1,,Me,https://www.linkedin.com/in/me,Ada,https://www.linkedin.com/in/Ada-L/,2026-05-02 09:00:00 UTC,,\"multi\nline\",INBOX\n" +
		"2,,Me,https://www.linkedin.com/in/me,\"Grace, Bob\",\"https://www.linkedin.com/in/grace,https://www.linkedin.com/in/bob\",2025-12-31 23:59:59 UTC,,,INBOX\n" +
		"3,,Me,https://www.linkedin.com/in/me,X,https://www.linkedin.com/in/x,not a date,,,INBOX\n"
	got, err := ParseLastMessages(strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"https://www.linkedin.com/in/ada-l": "2026-05-02", "https://www.linkedin.com/in/me": "2026-05-02",
		"https://www.linkedin.com/in/grace": "2025-12-31", "https://www.linkedin.com/in/bob": "2025-12-31",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v", got)
	}
}

func TestCanonicalURL(t *testing.T) {
	for in, want := range map[string]string{
		"http://linkedin.com/in/Foo%20Bar/": "https://www.linkedin.com/in/foo%20bar",
		"https://uk.linkedin.com/in/foo":    "https://www.linkedin.com/in/foo",
		"https://example.com/me":            "https://example.com/me",
		"javascript:alert(1)":               "",
		"":                                  "",
	} {
		if got := CanonicalURL(in); got != want {
			t.Errorf("CanonicalURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMatcher(t *testing.T) {
	m := NewMatcher([]Company{{"amazon", "Amazon"}, {"grafana-labs", "Grafana Labs"}, {"new-relic", "New Relic"}, {"company/cockroach", "Cockroach Labs"}})
	for in, want := range map[string]string{
		"Grafana Labs, Inc.":        "grafana-labs",
		"Amazon Web Services (AWS)": "amazon",
		"amazon":                    "amazon",
		"NEW RELIC":                 "new-relic",
		"Cockroach":                 "company/cockroach",
		"Amazonian Corp":            "",
		"":                          "",
	} {
		if got := m.Match(in); got != want {
			t.Errorf("Match(%q) = %q, want %q", in, got, want)
		}
	}
}
