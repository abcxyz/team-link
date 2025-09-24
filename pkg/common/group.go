// Copyright 2022 Google LLC
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
	"fmt"

	api "github.com/abcxyz/team-link/v2/apis/v1alpha3/proto"
	tltypes "github.com/abcxyz/team-link/v2/internal"
	googlegroupgithub "github.com/abcxyz/team-link/v2/pkg/common/googlegroup_github"
	"github.com/abcxyz/team-link/v2/pkg/groupsync"
)

// NewBidirectionalOneToManyGroupMapper creates two OneToManyGroupMapper, directions are src->target and target->src.
func NewBidirectionalOneToManyGroupMapper(source, target string, gm *api.GroupMappings, config *api.TeamLinkConfig) (groupsync.OneToManyGroupMapper, groupsync.OneToManyGroupMapper, error) {
	if source == tltypes.SystemTypeGoogleGroups && target == tltypes.SystemTypeGitHub {
		m := googlegroupgithub.NewBidirectionalGroupMapper(gm)
		return m.SourceMapper, m.TargetMapper, nil
	}
	return nil, nil, fmt.Errorf("unsupported sync flow from source system: %s to target system: %s", source, target)
}
