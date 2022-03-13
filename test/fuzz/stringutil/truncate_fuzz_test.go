package stringutil

import (
	"testing"
	"unicode/utf8"

	util "github.com/kube-tarian/tarian/pkg/stringutil"
)

func FuzzTruncate(f *testing.F) {
	f.Add("too long", 2)

	f.Fuzz(func(t *testing.T, str string, length int) {
		result := util.Truncate(str, length)
		if utf8.ValidString(str) && !utf8.ValidString(result) {
			t.Errorf("Truncate produced invalid UTF-8 string %q => %q", str, result)
		}

		if length >= 0 && utf8.RuneCountInString(result) > length {
			t.Errorf("Truncate produced string longer than expected length=%d, %q => %q", length, str, result)
		}
	})
}
