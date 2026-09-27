package leetgrinder

import (
	"bytes"
	"strings"
	"testing"
)

func TestAPITokens(t *testing.T) {
	a, hashA, err := NewAPIToken()
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := NewAPIToken()
	if err != nil {
		t.Fatal(err)
	}
	if a == b || !strings.HasPrefix(a, "lg_") || len(a) != 46 || !WellFormedAPIToken(a) {
		t.Fatalf("bad tokens %q %q", a, b)
	}
	if !bytes.Equal(hashA, HashAPIToken(a)) || len(hashA) != 32 {
		t.Fatal("hash mismatch")
	}
	for _, bad := range []string{"", "lg_", a[:45], a + "A", "xx_" + a[3:], "lg_" + strings.Repeat("!", 43)} {
		if WellFormedAPIToken(bad) {
			t.Errorf("accepted %q", bad)
		}
	}
	for name, ok := range map[string]bool{"Laptop": true, " Firefox ": true, "": false, strings.Repeat("é", 64): true, strings.Repeat("é", 65): false, "a\nb": false} {
		if _, err := ValidateAPITokenName(name); (err == nil) != ok {
			t.Errorf("%q: %v", name, err)
		}
	}
}
