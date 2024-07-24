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

package github

import (
	"github.com/google/go-github/v61/github"

	"github.com/abcxyz/team-link/pkg/paging"
)

type gitHubPageRequest struct {
	github.ListOptions
}

func (g *gitHubPageRequest) Opt() *github.ListOptions {
	return &g.ListOptions
}

func (g *gitHubPageRequest) Next() paging.PageRequest[*github.ListOptions] {
	g.Page++
	return g
}

type gitHubPage[T any] struct {
	content []T
	resp    *github.Response
}

func (g *gitHubPage[T]) Content() []T {
	return g.content
}

func (g *gitHubPage[T]) HasNext() bool {
	return g.resp.NextPage != 0
}
