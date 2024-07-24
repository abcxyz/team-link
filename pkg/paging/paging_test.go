// Copyright 2024 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package paging

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abcxyz/pkg/testutil"
)

type testPageRequest struct {
	number int
}

func (t *testPageRequest) Opt() int {
	return t.number
}

func (t *testPageRequest) Next() PageRequest[int] {
	t.number++
	return t
}

type testPage struct {
	content    []int
	number     int
	totalPages int
}

func (t *testPage) Content() []int {
	return t.content
}

func (t *testPage) HasNext() bool {
	return t.number+1 < t.totalPages
}

func TestPaginate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		pageData [][]int
		pager    Pager[int, int]
		want     []int
		wantErr  string
	}{
		{
			name:     "no_items",
			pageData: [][]int{{}},
			want:     []int{},
		},
		{
			name: "items_across_one_page",
			pageData: [][]int{
				{1, 2, 3},
			},
			want: []int{1, 2, 3},
		},
		{
			name: "items_across_several_pages",
			pageData: [][]int{
				{1, 2, 3},
				{4, 5, 6},
				{7, 8, 9},
			},
			want: []int{1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
		{
			name:     "error_when_paging",
			pageData: [][]int{{}},
			wantErr:  "pager error",
			want:     nil,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pager := func(request PageRequest[int]) (Page[int], error) {
				if tc.wantErr != "" {
					return nil, fmt.Errorf(tc.wantErr)
				}
				pageNumber := request.Opt()
				return &testPage{
					content:    tc.pageData[pageNumber],
					number:     pageNumber,
					totalPages: len(tc.pageData),
				}, nil
			}

			got, err := Paginate(&testPageRequest{number: 0}, pager)
			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("unexpected result (-got, +want):\n%s", diff)
			}
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error (-got, +want):\n%s", diff)
			}
		})
	}
}
