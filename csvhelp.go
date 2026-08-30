package main

import (
	"encoding/csv"
	"strings"
)

func csvReadAll(b []byte) ([][]string, error) {
	return csv.NewReader(strings.NewReader(string(b))).ReadAll()
}
