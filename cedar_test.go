package cedar

import (
	"testing"

	"github.com/vcaesar/tt"
)

var (
	cd *Cedar

	words = []string{
		"a", "aa", "ab", "abc", "abcd", "abcdef",
		"太阳系", "太阳系水星", "太阳系金星", "太阳系地球", "太阳系火星",
		"太阳系木星", "太阳系土星", "太阳系天王星", "太阳系海王星",
	}
)

func InitCd(reduced ...bool) error {
	cd = New(reduced...)
	return nil
}

func TestLoadData(t *testing.T) {
	if cd == nil {
		cd = New()
	}

	// insert the keys
	for i, word := range words {
		err := cd.Insert([]byte(word), i)
		tt.Nil(t, err)
	}

	for _, word := range words {
		err := cd.Delete([]byte(word))
		tt.Nil(t, err)
	}

	for i, word := range words {
		err := cd.Update([]byte(word), i)
		tt.Nil(t, err)
	}

	// delete the keys
	for i := 0; i < len(words); i += 3 {
		err := cd.Delete([]byte(words[i]))
		tt.Nil(t, err)
	}
}
