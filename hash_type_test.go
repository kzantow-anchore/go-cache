package cache_test

import (
	"fmt"
	"testing"

	"github.com/anchore/go-cache"
	"github.com/gohugoio/hashstructure"
	"github.com/stretchr/testify/require"
)

func Test_HashType(t *testing.T) {
	type t1 struct {
		Name string
	}
	type t2 struct {
		Name string
	}
	type generic[T any] struct {
		Val T
	}
	tests := []struct {
		name     string
		hash     func() string
		expected string
	}{
		{
			name:     "struct 1",
			hash:     func() string { return cache.HashType[t1]() },
			expected: "d106c3ffbf98a0b1",
		},
		{
			name:     "slice of struct 1",
			hash:     func() string { return cache.HashType[[]t1]() },
			expected: "8122ace4ee1af0b4",
		},
		{
			name:     "slice of struct 2",
			hash:     func() string { return cache.HashType[[]t2]() },
			expected: "8cc04b5808be5bf9",
		},
		{
			name:     "ptr 1",
			hash:     func() string { return cache.HashType[*t1]() },
			expected: "d106c3ffbf98a0b1", // same hash as t1, which is ok since the structs are the same
		},
		{
			name:     "slice of ptr 1",
			hash:     func() string { return cache.HashType[[]*t1]() },
			expected: "8122ace4ee1af0b4", // same hash as []t1, again underlying serialization is the same
		},
		{
			name:     "slice of ptr 2",
			hash:     func() string { return cache.HashType[[]*t2]() },
			expected: "8cc04b5808be5bf9", // same hash as []t2, underlying serialization is the same
		},
		{
			name:     "slice of ptr of slice of ptr",
			hash:     func() string { return cache.HashType[[]*[]*t1]() },
			expected: "500d9f5b3a5977ce",
		},
		{
			name:     "generic 1",
			hash:     func() string { return cache.HashType[generic[t1]]() },
			expected: "6c15f698eaa52524",
		},
		{
			name:     "generic 2",
			hash:     func() string { return cache.HashType[generic[t2]]() },
			expected: "858ec2607e33867d",
		},
		{
			name:     "generic with ptr 1",
			hash:     func() string { return cache.HashType[generic[*t1]]() },
			expected: "54988c656bc686f3",
		},
		{
			name:     "generic with ptr 2",
			hash:     func() string { return cache.HashType[generic[*t2]]() },
			expected: "5afd8dbbf5952c86",
		},
		{
			name:     "generic with slice 1",
			hash:     func() string { return cache.HashType[generic[[]t1]]() },
			expected: "e7cb716ac7101f7",
		},
		{
			name:     "generic with slice 2",
			hash:     func() string { return cache.HashType[generic[[]t2]]() },
			expected: "586f92ac7549d41",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.hash())
		})
	}
}

func Test_hashIgnores(t *testing.T) {
	hash := func(v any) string {
		v, err := hashstructure.Hash(v, &hashstructure.HashOptions{})
		require.NoError(t, err)
		return fmt.Sprintf("%x", v)
	}
	type t1 struct {
		Name        string
		notExported string
	}
	require.Equal(t, hash(t1{notExported: "a value"}), cache.HashType[t1]())

	type t2 struct {
		Name     string
		Exported string `hash:"ignore"`
	}
	require.Equal(t, hash(t2{Exported: "another value"}), cache.HashType[t2]())

	type t3 struct {
		Name     string
		Exported string `hash:"-"`
	}
	require.Equal(t, hash(t3{Exported: "still valued"}), cache.HashType[t3]())
}
