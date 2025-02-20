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
	"context"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/abcxyz/pkg/pointer"
)

// AccessLevelMapper provides a gitlab access level for a given group and user.
type AccessLevelMapper interface {
	// AccessLevel returns the gitlab access level for a user being added to a group.
	AccessLevel(ctx context.Context, groupID, userID string) *gitlab.AccessLevelValue
}

// DefaultAccessLevelMapper is the default AccessLevelMapper.
type DefaultAccessLevelMapper struct{}

// AccessLevel returns DeveloperPermissions for everything by default.
func (m *DefaultAccessLevelMapper) AccessLevel(ctx context.Context, groupID, userID string) *gitlab.AccessLevelValue {
	// By default, everyone gets Developer permissions.
	return pointer.To(gitlab.DeveloperPermissions)
}
