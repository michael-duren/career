package database

import (
	"strconv"
	"strings"
	"testing"
)

func TestMigrationVersionsAreUnique(t *testing.T) {
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		t.Fatal(err)
	}
	versions := map[int]string{}
	for _, file := range files {
		version, err := strconv.Atoi(strings.SplitN(file.Name(), "_", 2)[0])
		if err != nil {
			t.Fatalf("invalid migration filename %s: %v", file.Name(), err)
		}
		if previous, ok := versions[version]; ok {
			t.Fatalf("migration version %d is shared by %s and %s", version, previous, file.Name())
		}
		versions[version] = file.Name()
	}
}
