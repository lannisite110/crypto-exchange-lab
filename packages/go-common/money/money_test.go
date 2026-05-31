package money

import "testing"

func TestParse(t *testing.T) {
	d, err := Parse("100.5")
	if err != nil {
		t.Fatal(err)
	}
	if Format(d) != "100.5" {
		t.Fatalf("got %s", Format(d))
	}
}

func TestParseInvalid(t *testing.T) {
	if _, err := Parse("not-a-number"); err == nil {
		t.Fatal("expected error")
	}
}
