package store

import "testing"

func TestListUpMigrations(t *testing.T) {
	t.Parallel()
	names, err := listUpMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 || names[0] != "000001_create_control_plane.up.sql" {
		t.Fatalf("names = %#v", names)
	}
}

func TestNewPostgresStoreNilPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = NewPostgresStore(nil, nil)
}
