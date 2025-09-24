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

package gitlab

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/abcxyz/team-link/v2/pkg/groupsync"
)

// AccessLevelMetadata holds an access level for a GitLab user being added to
// a target group.
type AccessLevelMetadata struct {
	AccessLevel gitlab.AccessLevelValue
}

// Combine calculates the access level for a Gitlab user being added to
// a target group by taking the maximum access level granted to the user
// via a mapping from a source group.
func (m *AccessLevelMetadata) Combine(other groupsync.MappingMetadata) groupsync.MappingMetadata {
	if other == nil {
		return m
	}
	otherMetadata, ok := other.(*AccessLevelMetadata)
	if !ok {
		return m
	}
	if m == nil {
		return otherMetadata
	}

	// Take maximum access level granted to the user
	var level gitlab.AccessLevelValue
	if m.AccessLevel > otherMetadata.AccessLevel {
		level = m.AccessLevel
	} else {
		level = otherMetadata.AccessLevel
	}
	return &AccessLevelMetadata{
		AccessLevel: level,
	}
}
