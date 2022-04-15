package main

import (
	"os"
	"strings"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Converts nested map from terminal output to nested dicts
func main() {
	dat, err := os.ReadFile("nested_maps.txt")
	check(err)
	text := string(dat)
	text = strings.ReplaceAll(text, "%!q(bool=true)", "true")
	text = strings.ReplaceAll(text, "]", "}")
	text = strings.ReplaceAll(text, "map[", "dict{")
	text = strings.ReplaceAll(text, " ", ",")

	err = os.WriteFile("nested_dicts.txt", []byte(text), 0644)
	check(err)
}
