package main

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	s := "I can't see a goddamn thing!"
	order := 2
	res, err := tokenize(order, s)
	if err != nil {
		t.Fatal(err)
	}
	expected := [][]string{
		[]string{"I", "can't", "see"},
		[]string{"can't", "see", "a"},
		[]string{"see", "a", "goddamn"},
		[]string{"a", "goddamn", "thing!"},
		[]string{"goddamn", "thing!", "\n"},
	}

	if len(res) != len(expected) {
		t.Fatalf("different lengths: %q - %q\n", res, expected)
	}

	for i := range res {
		for u := range res[i] {
			if res[i][u] != expected[i][u] {
				t.Fatalf("%q != %q\n", res[i][u], expected[i][u])
			}
		}
	}
}
