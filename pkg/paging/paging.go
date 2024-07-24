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

// Package paging defines a generic paging pattern.
package paging

import "fmt"

type PageRequest[O any] interface {
	Opt() O
	Next() PageRequest[O]
}

type Page[T any] interface {
	Content() []T
	HasNext() bool
}

type Pager[T, O any] func(request PageRequest[O]) (Page[T], error)

func Paginate[T, O any](pageRequest PageRequest[O], pager Pager[T, O]) ([]T, error) {
	items := make([]T, 0)
	for {
		page, err := pager(pageRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to get page: %w", err)
		}
		items = append(items, page.Content()...)
		if !page.HasNext() {
			break
		}
		pageRequest = pageRequest.Next()
	}

	return items, nil
}
