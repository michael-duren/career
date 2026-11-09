package leetgrinder

import (
	"errors"
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
	if _, err := ParseTodoRefs("two-sum\nhttps://example.com/problems/x\nthree-sum"); !errors.As(err, new(InvalidTodoRefError)) || !strings.Contains(err.Error(), "example.com") {
		t.Errorf("the error should name the bad entry: %v", err)
	}
	if got, err := ParseTodoRefs("two-sum\nhttps://neetcode.io/problems/heap"); err != nil || !reflect.DeepEqual(got, []string{"two-sum", "neetcode-heap"}) {
		t.Errorf("NeetCode-only entry: %v %v", got, err)
	}
	many := strings.Repeat("a-b ", MaxTodoRefs+1)
	if _, err := ParseTodoRefs(many); !errors.Is(err, ErrTooManyTodoRefs) {
		t.Errorf("too many: %v", err)
	}
}

func TestParseTodoRefsMixed(t *testing.T) {
	got, err := ParseTodoRefs("https://neetcode.io/problems/two-integer-sum https://leetcode.cn/problems/3sum/, www.leetcode.cn/problems/valid-anagram\nneetcode.io/problems/two-integer-sum/question two-sum contains-duplicate")
	want := []string{"two-sum", "3sum", "valid-anagram", "contains-duplicate"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("mixed refs: %v %v", got, err)
	}
}

func TestParseTodoRefsTruncatesByRune(t *testing.T) {
	_, err := ParseTodoRefs(strings.Repeat("é", 100))
	var invalid InvalidTodoRefError
	if !errors.As(err, &invalid) || invalid.Entry != strings.Repeat("é", 60)+"..." {
		t.Fatalf("entry %q", invalid.Entry)
	}
}
