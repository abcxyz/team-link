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

package client

import (
	"context"
	"fmt"

	tltypes "github.com/abcxyz/team-link/internal"
	ggtogh "github.com/abcxyz/team-link/pkg/client/googlegroup_github"
	"github.com/abcxyz/team-link/pkg/groupsync"
)

// NewOneToManyGroupMapper creates a groupsync.OneToManyMapper base on the input source
// and destination system type using provided groupMappingFile.
func NewBidirectionalNewOneToManyGroupMapper(source, dest, groupMappingFile string) (groupsync.OneToManyGroupMapper, groupsync.OneToManyGroupMapper, error) {
	if source == tltypes.SystemTypeGoogleGroups && dest == tltypes.SystemTypeGitHub {
		m, err := ggtogh.NewBidirectionaGroupMapper(groupMappingFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create group mapper for GoogleGroupToGitHub: %w", err)
		}
		return m.SourceMapper, m.TargetMapper, nil
	}
	return nil, nil, fmt.Errorf("unsupported source to dest mapper type: source %s, dest %s", source, dest)
}

// NewUserMapper creats a UserMapper base on source and dest system type.
func NewUserMapper(ctx context.Context, source, dest, mappingFilePath string) (groupsync.UserMapper, error) {
	if source == tltypes.SystemTypeGoogleGroups && dest == tltypes.SystemTypeGitHub {
		m, err := ggtogh.NewUserMapper(ctx, mappingFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create GoogleGroupGitHubUserMapper: %w", err)
		}
		return m, nil
	}
	return nil, fmt.Errorf("unsupported source to dest user mapper type: source %s, dest %s", source, dest)
}
