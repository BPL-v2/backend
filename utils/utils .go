package utils

import (
	"fmt"
	"log"
	"strings"

	"github.com/lib/pq"
)

func Map[A any, B any](input []A, mapper func(A) B) []B {
	output := make([]B, len(input))
	for i, item := range input {
		output[i] = mapper(item)
	}
	return output
}

// SplitEscaped splits s on commas and trims whitespace around each part.
// A comma can be kept as part of a value by escaping it with a backslash (`d\,e`),
// and a literal backslash can be written as `\\`. Any other backslash is kept as is.
func SplitEscaped(s string) []string {
	parts := []string{}
	var current strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) && (s[i+1] == ',' || s[i+1] == '\\') {
			current.WriteByte(s[i+1])
			i++
		} else if c == ',' {
			parts = append(parts, strings.TrimSpace(current.String()))
			current.Reset()
		} else {
			current.WriteByte(c)
		}
	}
	return append(parts, strings.TrimSpace(current.String()))
}

func FilterNull[A any](input []*A) []*A {
	output := make([]*A, 0)
	for _, item := range input {
		if item != nil {
			output = append(output, item)
		}
	}
	return output
}

func Reduce[A any](input []*A, reducer func(*A, *A) *A) *A {
	if len(input) == 0 {
		return nil
	}
	result := input[0]
	for _, item := range input[1:] {
		result = reducer(result, item)
	}
	return result
}

func FindFirst[A any](input []*A, predicate func(*A) bool) (*A, bool) {
	for _, item := range input {
		if predicate(item) {
			return item, true
		}
	}
	return nil, false
}

func FlatMap[A any, B any](input []A, mapper func(A) []B) []B {
	return Flatten(Map(input, mapper))
}

func Flatten[A any](input [][]A) []A {
	output := make([]A, 0)
	for _, item := range input {
		output = append(output, item...)
	}
	return output
}

func Filter[A any](input []A, filter func(A) bool) []A {
	output := make([]A, 0)
	for _, item := range input {
		if filter(item) {
			output = append(output, item)
		}
	}
	return output
}

func Keys[A comparable, B any](input map[A]B) []A {
	keys := make([]A, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	return keys
}

func Values[A comparable, B any](input map[A]B) []B {
	values := make([]B, 0, len(input))
	for _, value := range input {
		values = append(values, value)
	}
	return values
}

func Uniques[A comparable](input []A) []A {
	ids := make(map[A]bool)
	for _, item := range input {
		ids[item] = true
	}
	return Keys(ids)
}

func BatchIterator[A any](input []A, batchSize int) <-chan []A {
	ch := make(chan []A)
	go func() {
		defer close(ch)
		for i := 0; i < len(input); i += batchSize {
			end := min(i+batchSize, len(input))
			ch <- input[i:end]
		}
	}()
	return ch
}

type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string
}

func Max[T Ordered](a ...T) T {
	if len(a) == 0 {
		var zero T
		return zero
	}
	max := a[0]
	for _, v := range a {
		if v > max {
			max = v
		}
	}
	return max
}

func Min[T Ordered](a ...T) T {
	if len(a) == 0 {
		var zero T
		return zero
	}
	min := a[0]
	for _, v := range a {
		if v < min {
			min = v
		}
	}
	return min
}

// mimic the python datatype
type Set[T comparable] map[T]bool

func ToSet[T comparable](a []T) Set[T] {
	set := make(map[T]bool)
	for _, v := range a {
		set[v] = true
	}
	return set
}

func (s Set[T]) String() string {
	parts := make([]string, 0, len(s))
	for k := range s {
		parts = append(parts, fmt.Sprintf("%v", k))
	}
	return "set[" + strings.Join(parts, ", ") + "]"
}

func (s Set[T]) ToSlice() []T {
	slice := make([]T, 0, len(s))
	for k := range s {
		slice = append(slice, k)
	}
	return slice
}

func (s Set[T]) Add(items ...T) {
	for _, item := range items {
		s[item] = true
	}
}

func (s1 Set[T]) Intersection(s2 Set[T]) Set[T] {
	intersection := make(map[T]bool)
	for k := range s1 {
		if _, ok := s2[k]; ok {
			intersection[k] = true
		}
	}
	return intersection
}

func (s1 Set[T]) Difference(s2 Set[T]) Set[T] {
	difference := make(map[T]bool)
	for k := range s1 {
		if _, ok := s2[k]; !ok {
			difference[k] = true
		}
	}
	return difference
}

func (s1 Set[T]) Union(s2 Set[T]) Set[T] {
	union := make(map[T]bool)
	for k := range s1 {
		union[k] = true
	}
	for k := range s2 {
		union[k] = true
	}
	return union
}

func ConvertIntSlice(slice []int) pq.Int32Array {
	arr := pq.Int32Array{}
	for _, integer := range slice {
		arr = append(arr, int32(integer))
	}
	return arr
}

func Deref[T any](value *T) T {
	var zero T
	if value == nil {
		return zero
	}
	return *value
}

func ArrayEquals[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func ArrayEqualsUnordered[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[T]int)
	for _, item := range a {
		counts[item]++
	}
	for _, item := range b {
		count, exists := counts[item]
		if !exists || count == 0 {
			return false
		}
		counts[item]--
	}
	return true
}

type Closable interface {
	Close() error
}

func Closer(c Closable) func() {
	return func() {
		if err := c.Close(); err != nil {
			log.Printf("Error closing resource: %v\n", err)
		}
	}
}
