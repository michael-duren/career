package leetgrinder

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseTodoRefs(t *testing.T) {
	want := []string{"two-sum", "three-sum", "valid-anagram"}
	for _, input := range []string{
		"two-sum\n\nthree-sum\nvalid-anagram",
		"two-sum, three-sum, valid-anagram, ",
		"two-sum\n   \nthree-sum\r\nvalid-anagram\n",
		"two-sum three-sum\tvalid-anagram",
		"https://leetcode.com/problems/two-sum/, three-sum\ntwo-sum\nvalid-anagram",
	} {
		got, err := ParseTodoRefs(input)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%q: %v %v", input, got, err)
		}
	}
	if got, err := ParseTodoRefs(" \n,\n"); err != nil || len(got) != 0 {
		t.Errorf("blank list: %v %v", got, err)
	}
	if _, err := ParseTodoRefs("two-sum\nhttps://example.com/problems/x\nthree-sum"); err == nil || !strings.Contains(err.Error(), "example.com") {
		t.Errorf("the error should name the bad entry: %v", err)
	}
	many := strings.Repeat("a-b ", MaxTodoRefs+1)
	if _, err := ParseTodoRefs(many); err == nil || !strings.Contains(err.Error(), "at most 200") {
		t.Errorf("too many: %v", err)
	}
}
