package fbclient

import "testing"

func TestHumanToBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"0", 0, false},
		{"", 0, false},
		{"100", 100, false},
		{"1K", 1024, false},
		{"1k", 1024, false},
		{"10M", 10 * 1024 * 1024, false},
		{"100G", 100 * 1024 * 1024 * 1024, false},
		{"1T", 1024 * 1024 * 1024 * 1024, false},
		{"2P", 2 * 1024 * 1024 * 1024 * 1024 * 1024, false},
		{"500M", 500 * 1024 * 1024, false},
		{"abc", 0, true},
		{"10X", 0, true},
	}

	for _, tt := range tests {
		result, err := HumanToBytes(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("HumanToBytes(%q) expected error, got %d", tt.input, result)
			}
			continue
		}
		if err != nil {
			t.Errorf("HumanToBytes(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("HumanToBytes(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestBytesToHuman(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, ""},
		{-1, ""},
		{100, "100"},
		{1024, "1K"},
		{10 * 1024 * 1024, "10M"},
		{100 * 1024 * 1024 * 1024, "100G"},
		{1024 * 1024 * 1024 * 1024, "1T"},
	}

	for _, tt := range tests {
		result := BytesToHuman(tt.input)
		if result != tt.expected {
			t.Errorf("BytesToHuman(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
