package utils

import (
	"errors"
	"log"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== Map ====================

func TestMap_IntToString(t *testing.T) {
	result := Map([]int{1, 2, 3}, func(i int) string { return string(rune('A' + i - 1)) })
	assert.Equal(t, []string{"A", "B", "C"}, result)
}

func TestMap_Empty(t *testing.T) {
	result := Map([]int{}, func(i int) int { return i * 2 })
	assert.Empty(t, result)
}

func TestMap_Double(t *testing.T) {
	result := Map([]int{1, 2, 3}, func(i int) int { return i * 2 })
	assert.Equal(t, []int{2, 4, 6}, result)
}

// ==================== Reduce ====================

func TestReduce_Sum(t *testing.T) {
	a, b, c := 1, 2, 3
	result := Reduce([]*int{&a, &b, &c}, func(acc *int, item *int) *int {
		sum := *acc + *item
		return &sum
	})
	assert.Equal(t, 6, *result)
}

func TestReduce_Empty(t *testing.T) {
	result := Reduce([]*int{}, func(acc *int, item *int) *int { return acc })
	assert.Nil(t, result)
}

func TestReduce_Single(t *testing.T) {
	val := 42
	result := Reduce([]*int{&val}, func(acc *int, item *int) *int { return acc })
	assert.Equal(t, 42, *result)
}

// ==================== FindFirst ====================

func TestFindFirst_Found(t *testing.T) {
	a, b, c := 1, 2, 3
	result, found := FindFirst([]*int{&a, &b, &c}, func(i *int) bool { return *i == 2 })
	assert.True(t, found)
	assert.Equal(t, 2, *result)
}

func TestFindFirst_NotFound(t *testing.T) {
	a, b := 1, 3
	result, found := FindFirst([]*int{&a, &b}, func(i *int) bool { return *i == 2 })
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestFindFirst_Empty(t *testing.T) {
	result, found := FindFirst([]*int{}, func(i *int) bool { return true })
	assert.False(t, found)
	assert.Nil(t, result)
}

// ==================== FlatMap ====================

func TestFlatMap(t *testing.T) {
	result := FlatMap([]string{"hello", "world"}, func(s string) []string {
		return strings.Split(s, "")
	})
	assert.Equal(t, []string{"h", "e", "l", "l", "o", "w", "o", "r", "l", "d"}, result)
}

func TestFlatMap_Empty(t *testing.T) {
	result := FlatMap([]int{}, func(i int) []int { return []int{i, i} })
	assert.Empty(t, result)
}

// ==================== Flatten ====================

func TestFlatten(t *testing.T) {
	result := Flatten([][]int{{1, 2}, {3, 4}, {5}})
	assert.Equal(t, []int{1, 2, 3, 4, 5}, result)
}

func TestFlatten_Empty(t *testing.T) {
	result := Flatten([][]int{})
	assert.Empty(t, result)
}

func TestFlatten_WithEmptyInner(t *testing.T) {
	result := Flatten([][]int{{1}, {}, {3}})
	assert.Equal(t, []int{1, 3}, result)
}

// ==================== Filter ====================

func TestFilter_Even(t *testing.T) {
	result := Filter([]int{1, 2, 3, 4, 5}, func(i int) bool { return i%2 == 0 })
	assert.Equal(t, []int{2, 4}, result)
}

func TestFilter_None(t *testing.T) {
	result := Filter([]int{1, 3, 5}, func(i int) bool { return i%2 == 0 })
	assert.Empty(t, result)
}

func TestFilter_All(t *testing.T) {
	result := Filter([]int{2, 4, 6}, func(i int) bool { return i%2 == 0 })
	assert.Equal(t, []int{2, 4, 6}, result)
}

func TestFilter_Empty(t *testing.T) {
	result := Filter([]int{}, func(i int) bool { return true })
	assert.Empty(t, result)
}

// ==================== Keys ====================

func TestKeys(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	keys := Keys(m)
	sort.Strings(keys)
	assert.Equal(t, []string{"a", "b", "c"}, keys)
}

func TestKeys_Empty(t *testing.T) {
	m := map[string]int{}
	keys := Keys(m)
	assert.Empty(t, keys)
}

// ==================== Values ====================

func TestValues(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	vals := Values(m)
	sort.Ints(vals)
	assert.Equal(t, []int{1, 2}, vals)
}

func TestValues_Empty(t *testing.T) {
	m := map[string]int{}
	vals := Values(m)
	assert.Empty(t, vals)
}

// ==================== Uniques ====================

func TestUniques(t *testing.T) {
	result := Uniques([]int{1, 2, 2, 3, 1, 4})
	sort.Ints(result)
	assert.Equal(t, []int{1, 2, 3, 4}, result)
}

func TestUniques_AllSame(t *testing.T) {
	result := Uniques([]int{5, 5, 5})
	assert.Equal(t, []int{5}, result)
}

func TestUniques_Empty(t *testing.T) {
	result := Uniques([]int{})
	assert.Empty(t, result)
}

func TestUniques_Strings(t *testing.T) {
	result := Uniques([]string{"a", "b", "a"})
	sort.Strings(result)
	assert.Equal(t, []string{"a", "b"}, result)
}

// ==================== BatchIterator ====================

func TestBatchIterator_Normal(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}
	var batches [][]int
	for batch := range BatchIterator(input, 2) {
		batches = append(batches, batch)
	}
	assert.Len(t, batches, 3)
	assert.Equal(t, []int{1, 2}, batches[0])
	assert.Equal(t, []int{3, 4}, batches[1])
	assert.Equal(t, []int{5}, batches[2])
}

func TestBatchIterator_ExactBatch(t *testing.T) {
	input := []int{1, 2, 3, 4}
	var batches [][]int
	for batch := range BatchIterator(input, 2) {
		batches = append(batches, batch)
	}
	assert.Len(t, batches, 2)
}

func TestBatchIterator_Empty(t *testing.T) {
	var batches [][]int
	for batch := range BatchIterator([]int{}, 2) {
		batches = append(batches, batch)
	}
	assert.Empty(t, batches)
}

func TestBatchIterator_SingleBatch(t *testing.T) {
	input := []int{1, 2, 3}
	var batches [][]int
	for batch := range BatchIterator(input, 10) {
		batches = append(batches, batch)
	}
	assert.Len(t, batches, 1)
	assert.Equal(t, []int{1, 2, 3}, batches[0])
}

// ==================== Max ====================

func TestMax_Ints(t *testing.T) {
	assert.Equal(t, 5, Max(1, 5, 3))
}

func TestMax_Single(t *testing.T) {
	assert.Equal(t, 42, Max(42))
}

func TestMax_Negative(t *testing.T) {
	assert.Equal(t, -1, Max(-3, -1, -2))
}

func TestMax_Empty(t *testing.T) {
	assert.Equal(t, 0, Max[int]())
}

func TestMax_Strings(t *testing.T) {
	assert.Equal(t, "z", Max("a", "z", "m"))
}

// ==================== Min ====================

func TestMin_Ints(t *testing.T) {
	assert.Equal(t, 1, Min(1, 5, 3))
}

func TestMin_Single(t *testing.T) {
	assert.Equal(t, 42, Min(42))
}

func TestMin_Negative(t *testing.T) {
	assert.Equal(t, -3, Min(-3, -1, -2))
}

func TestMin_Empty(t *testing.T) {
	assert.Equal(t, 0, Min[int]())
}

func TestMin_Strings(t *testing.T) {
	assert.Equal(t, "a", Min("a", "z", "m"))
}

// ==================== Set ====================

func TestToSet(t *testing.T) {
	set := ToSet([]int{1, 2, 3, 2, 1})
	assert.Len(t, set, 3)
	assert.True(t, set[1])
	assert.True(t, set[2])
	assert.True(t, set[3])
}

func TestToSet_Empty(t *testing.T) {
	set := ToSet([]int{})
	assert.Empty(t, set)
}

func TestSet_Intersection(t *testing.T) {
	s1 := ToSet([]int{1, 2, 3})
	s2 := ToSet([]int{2, 3, 4})
	result := s1.Intersection(s2)
	assert.Len(t, result, 2)
	assert.True(t, result[2])
	assert.True(t, result[3])
}

func TestSet_Intersection_Empty(t *testing.T) {
	s1 := ToSet([]int{1, 2})
	s2 := ToSet([]int{3, 4})
	result := s1.Intersection(s2)
	assert.Empty(t, result)
}

func TestSet_Difference(t *testing.T) {
	s1 := ToSet([]int{1, 2, 3})
	s2 := ToSet([]int{2, 3, 4})
	result := s1.Difference(s2)
	assert.Len(t, result, 1)
	assert.True(t, result[1])
}

func TestSet_Difference_Empty(t *testing.T) {
	s1 := ToSet([]int{1, 2})
	s2 := ToSet([]int{1, 2, 3})
	result := s1.Difference(s2)
	assert.Empty(t, result)
}

func TestSet_Union(t *testing.T) {
	s1 := ToSet([]int{1, 2, 3})
	s2 := ToSet([]int{3, 4, 5})
	result := s1.Union(s2)
	assert.Len(t, result, 5)
	for i := 1; i <= 5; i++ {
		assert.True(t, result[i])
	}
}

func TestSet_Union_Empty(t *testing.T) {
	s1 := ToSet([]int{})
	s2 := ToSet([]int{1})
	result := s1.Union(s2)
	assert.Len(t, result, 1)
}

// ==================== ConvertIntSlice ====================

func TestConvertIntSlice(t *testing.T) {
	result := ConvertIntSlice([]int{1, 2, 3})
	assert.Len(t, result, 3)
	assert.Equal(t, int32(1), result[0])
	assert.Equal(t, int32(2), result[1])
	assert.Equal(t, int32(3), result[2])
}

func TestConvertIntSlice_Empty(t *testing.T) {
	result := ConvertIntSlice([]int{})
	assert.Empty(t, result)
}

// ==================== Deref ====================

func TestDeref_NonNil(t *testing.T) {
	val := 42
	assert.Equal(t, 42, Deref(&val))
}

func TestDeref_Nil(t *testing.T) {
	var p *int
	assert.Equal(t, 0, Deref(p))
}

func TestDeref_String(t *testing.T) {
	s := "hello"
	assert.Equal(t, "hello", Deref(&s))
}

func TestDeref_NilString(t *testing.T) {
	var s *string
	assert.Equal(t, "", Deref(s))
}

// ==================== ArrayEquals ====================

func TestArrayEquals_Equal(t *testing.T) {
	assert.True(t, ArrayEquals([]int{1, 2, 3}, []int{1, 2, 3}))
}

func TestArrayEquals_DifferentLength(t *testing.T) {
	assert.False(t, ArrayEquals([]int{1, 2}, []int{1, 2, 3}))
}

func TestArrayEquals_DifferentContent(t *testing.T) {
	assert.False(t, ArrayEquals([]int{1, 2, 3}, []int{1, 3, 2}))
}

func TestArrayEquals_Empty(t *testing.T) {
	assert.True(t, ArrayEquals([]int{}, []int{}))
}

func TestArrayEquals_Strings(t *testing.T) {
	assert.True(t, ArrayEquals([]string{"a", "b"}, []string{"a", "b"}))
	assert.False(t, ArrayEquals([]string{"a", "b"}, []string{"b", "a"}))
}

// ==================== ArrayEqualsUnordered ====================

func TestArrayEqualsUnordered_Equal(t *testing.T) {
	assert.True(t, ArrayEqualsUnordered([]int{1, 2, 3}, []int{3, 2, 1}))
}

func TestArrayEqualsUnordered_DifferentLength(t *testing.T) {
	assert.False(t, ArrayEqualsUnordered([]int{1, 2}, []int{1, 2, 3}))
}

func TestArrayEqualsUnordered_DifferentContent(t *testing.T) {
	assert.False(t, ArrayEqualsUnordered([]int{1, 2, 3}, []int{1, 2, 4}))
}

func TestArrayEqualsUnordered_Empty(t *testing.T) {
	assert.True(t, ArrayEqualsUnordered([]int{}, []int{}))
}

func TestArrayEqualsUnordered_DuplicatesMatch(t *testing.T) {
	assert.True(t, ArrayEqualsUnordered([]int{1, 1, 2}, []int{2, 1, 1}))
}

func TestArrayEqualsUnordered_DuplicatesMismatch(t *testing.T) {
	assert.False(t, ArrayEqualsUnordered([]int{1, 1, 2}, []int{1, 2, 2}))
}

// ==================== Closer ====================

type mockClosable struct {
	err error
}

func (m *mockClosable) Close() error {
	return m.err
}

func TestCloser_Success(t *testing.T) {
	c := &mockClosable{err: nil}
	closer := Closer(c)
	closer() // Should not panic
}

func TestCloser_Error(t *testing.T) {
	c := &mockClosable{err: errors.New("close error")}
	closer := Closer(c)
	// Capture log output
	var buf strings.Builder
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	closer()
	assert.Contains(t, buf.String(), "Error closing resource")
}

// ==================== SplitEscaped ====================

func TestSplitEscaped(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c"}, SplitEscaped("a, b,c"))
	assert.Equal(t, []string{"d,e", "b", "c"}, SplitEscaped(`d\,e,b, c`))
	assert.Equal(t, []string{`a\`, "b"}, SplitEscaped(`a\\,b`))
	assert.Equal(t, []string{`a\x`}, SplitEscaped(`a\x`))
	assert.Equal(t, []string{"single"}, SplitEscaped("single"))
}
