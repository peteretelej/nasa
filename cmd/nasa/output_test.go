package main

import "testing"

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short_string", "hello", 10, "hello"},
		{"exact_length", "hello", 5, "hello"},
		{"truncated", "hello world", 8, "hello..."},
		{"maxlen_3", "hello", 3, "hel"},
		{"maxlen_2", "hello", 2, "he"},
		{"maxlen_1", "hello", 1, "h"},
		{"maxlen_0", "hello", 0, ""},
		{"maxlen_negative", "hello", -1, ""},
		{"empty_string", "", 5, ""},
		{"empty_string_zero", "", 0, ""},
		// Multi-byte UTF-8 strings.
		{"cjk_no_truncate", "\u4f60\u597d\u4e16\u754c", 4, "\u4f60\u597d\u4e16\u754c"},
		{"cjk_truncate", "\u4f60\u597d\u4e16\u754c\u4eba", 4, "\u4f60..."},
		{"emoji_no_truncate", "\U0001f600\U0001f601\U0001f602", 3, "\U0001f600\U0001f601\U0001f602"},
		{"emoji_truncate", "\U0001f600\U0001f601\U0001f602\U0001f603", 3, "\U0001f600\U0001f601\U0001f602"},
		{"emoji_truncate_with_ellipsis", "\U0001f600\U0001f601\U0001f602\U0001f603\U0001f604", 4, "\U0001f600..."},
		{"mixed_multibyte", "a\u4f60b\u597dc", 4, "a..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}
