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

package common

import (
	"context"
	"fmt"

	api "github.com/abcxyz/team-link/v2/apis/v1alpha3/proto"
	tltypes "github.com/abcxyz/team-link/v2/internal"
	gggh "github.com/abcxyz/team-link/v2/pkg/common/googlegroup_github"
	"github.com/abcxyz/team-link/v2/pkg/groupsync"
)

// NewUserMapper creates a new UserMapper base on source and target system type.
func NewUserMapper(ctx context.Context, source, target string, mappings *api.UserMappings) (groupsync.UserMapper, error) {
	if source == tltypes.SystemTypeGoogleGroups && target == tltypes.SystemTypeGitHub {
		m := gggh.NewUserMapper(ctx, mappings)
		return m, nil
	}
	return nil, fmt.Errorf("unsupported source to dest user mapper type: source %s, dest %s", source, target)
}
