package init

import "testing"

func TestValidateHistory(t *testing.T) {
	chain := []embeddedMigration{{id: "00000000"}, {id: "00010000"}, {id: "00020000"}}
	tests := []struct {
		name string
		rows [][]string
		want int
		bad  bool
	}{
		{"prefix unordered", [][]string{{"00010000"}, {"00000000"}}, 2, false},
		{"complete", [][]string{{"00020000"}, {"00000000"}, {"00010000"}}, 3, false},
		{"gap", [][]string{{"00000000"}, {"00020000"}}, 0, true},
		{"unknown", [][]string{{"00000000"}, {"99999999"}}, 0, true},
		{"empty", nil, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateHistory("spec", chain, tt.rows)
			if (err != nil) != tt.bad || got != tt.want {
				t.Fatalf("got %d, %v", got, err)
			}
		})
	}
}

func TestParseIdentityAnswer(t *testing.T) {
	for _, tc := range []struct {
		answer    string
		self, bad bool
	}{
		{"4sight\n", true, false}, {"application\n", false, false},
		{"4sight\napplication\n", false, true}, {"4sight", false, true}, {"", false, true},
	} {
		self, err := parseIdentityAnswer(tc.answer)
		if self != tc.self || (err != nil) != tc.bad {
			t.Fatalf("%q: self=%t error=%v", tc.answer, self, err)
		}
	}
}
