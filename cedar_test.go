package cedar

import (
	"fmt"
	"testing"

	"github.com/vcaesar/tt"
)

var (
	cd *Cedar

	words = []string{
		"a", "aa", "ab", "abc", "abcd", "abcdef",
		"太阳系", "太阳系水星", "太阳系金星", "太阳系地球", "太阳系火星",
		"太阳系木星", "太阳系土星", "太阳系天王星", "太阳系海王星",
		"this", "this is", "this is a cedar.",
	}
)

func InitCd(reduced ...bool) error {
	cd = New(reduced...)
	return nil
}

func TestLoadData(t *testing.T) {
	// add the words
	for i, word := range words {
		err := cd.Insert([]byte(word), i)
		tt.Nil(t, err)
	}

	// update the words
	for i, word := range words {
		err := cd.Delete([]byte(word))
		tt.Nil(t, err)

		err = cd.Update([]byte(word), i)
		tt.Nil(t, err)
	}

	// delete not used word
	for i := 10; i < 15; i++ {
		err := cd.Delete([]byte(words[i]))
		tt.Nil(t, err)
	}
}

func TestFind(t *testing.T) {

	key, err := cd.Find([]byte("a"), 0)
	tt.Nil(t, err)
	tt.Equal(t, 0, key)

	val, err := cd.Get([]byte("ab"))
	tt.Nil(t, err)
	tt.Equal(t, 2, val)

	to, err := cd.Jump([]byte("abc"), 0)
	tt.Nil(t, err)
	tt.Equal(t, 352, to)
	val, err = cd.Value(to)
	tt.Nil(t, err)
	tt.Equal(t, 3, val)
}

func TestPrefixMatch(t *testing.T) {
	ids := cd.PrefixMatch([]byte("this is a cedar."), 0)
	fmt.Println("ids: ", ids)
	keys := []string{"this", "this is", "this is a cedar."}
	values := []int{15, 16, 17}
	tt.Equal(t, len(keys), len(ids))
	for i, n := range ids {
		v, _ := cd.Value(n)
		tt.Equal(t, values[i], v)
	}
}

func TestPrefixPredict(t *testing.T) {
	ids := cd.PrefixPredict([]byte("太阳系"), 0)
	fmt.Println("ids: ", ids)

	keys := []string{"太阳系", "太阳系地球", "太阳系水星", "太阳系金星"}
	values := []int{6, 9, 7, 8}
	tt.Equal(t, len(keys), len(ids))
	for i, n := range ids {
		v, _ := cd.Value(n)
		tt.Equal(t, values[i], v)
	}
}
