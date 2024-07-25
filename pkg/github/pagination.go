package github

import (
	"cmp"
	"fmt"
	"sort"

	"github.com/google/go-github/v61/github"
)

func Paginate[T cmp.Ordered](f func(x func(T), opts *github.ListOptions) (*github.Response, error)) ([]T, error) {
	tm := make(map[T]struct{}, 32)

	accum := func(i T) {
		if _, ok := tm[i]; !ok {
			tm[i] = struct{}{}
		}
	}

	opts := &github.ListOptions{
		PerPage: 100,
	}

	for {
		resp, err := f(accum, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to paginate: %w", err)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	t := make([]T, 0, len(tm))
	for k := range tm {
		t = append(t, k)
	}
	sort.Slice(t, func(i, j int) bool {
		return t[i] < t[j]
	})

	return t, nil
}
