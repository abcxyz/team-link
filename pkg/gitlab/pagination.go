// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package gitlab

import (
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// paginate is a helper function that iterates through a series of
// well-structured GitLab responses by continuously invoking `f` for each
// `NextPage` token. It is the caller's responsibility to capture any values
// inside the closer (e.g. append to a slice or map); this function does not
// accumulate responses.
func paginate(f func(opts *gitlab.ListOptions) (*gitlab.Response, error)) error {
	opts := &gitlab.ListOptions{
		PerPage: 100,
	}

	for {
		resp, err := f(opts)
		if err != nil {
			return fmt.Errorf("failed to paginate: %w", err)
		}

		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return nil
}
