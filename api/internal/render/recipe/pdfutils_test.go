package recipe

import (
	"reflect"
	"testing"
)

func TestParsePageRanges(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"whitespace only", "   ", nil},
		{"single page", "2", []string{"2"}},
		{"range", "5-7", []string{"5-7"}},
		{"mixed with spaces", "2, 5-7, last", []string{"2", "5-7", "l"}},
		{"last lowercase", "last", []string{"l"}},
		{"last uppercase", "LAST", []string{"l"}},
		{"last mixed case", "LaSt", []string{"l"}},
		{"trailing comma", "1,2,", []string{"1", "2"}},
		{"empty segments", "1,,3", []string{"1", "3"}},
		{"lots of whitespace", "  1  ,  2  ", []string{"1", "2"}},
		{"last among numbers", "1, last, 3", []string{"1", "l", "3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePageRanges(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePageRanges(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
