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
	"os"
	"slices"

	"google.golang.org/protobuf/encoding/prototext"

	"github.com/abcxyz/team-link/apis/v1alpha3"
	tltypes "github.com/abcxyz/team-link/internal"
	ggtogh "github.com/abcxyz/team-link/pkg/client/googlegroup_github"
	"github.com/abcxyz/team-link/pkg/github"
	"github.com/abcxyz/team-link/pkg/groupsync"
)

// GroupMapper implements groupsync.OneToManyGroupMapper
// For now it was be used as Mapper between GoogleGroup
// and GitHubTeam.
// Both GoogleGroupMapper and GitHubMapper have similar
// variables and functions, this way we don't need to repeatly
// write similar function for both on them.
type GroupMapper map[string][]string

func (m GroupMapper) AllGroupIDs(ctx context.Context) ([]string, error) {
	res := make([]string, 0)
	for key := range m {
		res = append(res, key)
	}
	slices.Sort(res)
	return res, nil
}

func (m GroupMapper) ContainsGroupID(ctx context.Context, key string) (bool, error) {
	_, ok := m[key]
	if !ok {
		return false, fmt.Errorf("group %s is not mapped", key)
	}
	return ok, nil
}

func (m GroupMapper) MappedGroupIDs(ctx context.Context, key string) ([]string, error) {
	x, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("no mapping found for group ID: %s", key)
	}
	return x, nil
}

type GoogleGroupToGitHubMapper GroupMapper

type GitHubToGoogleGroupMapper GroupMapper

// NewBidirectionalGoogleGroupGitHubMapper creates a GoogleGroupToGitHubMapper
// and a GitHubToGoogleGroupMapper using the provided groupMapping file.
// Returns is (GoogleGroupToGitHubMapper, GitHubToGoogleGroupMapper, error).
//
// TODO: refactor this into client/googlegroup_github/mapper.go later
func NewBidirectionalGoogleGroupGitHubMapper(groupMappingFile string) (groupsync.OneToManyGroupMapper, groupsync.OneToManyGroupMapper, error) {
	b, err := os.ReadFile(groupMappingFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read mapping file: %w", err)
	}
	tm := &v1alpha3.GoogleGroupToGitHubTeamMappings{}
	if err := prototext.Unmarshal(b, tm); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal mapping file: %w", err)
	}
	ggToGHMapping := make(GroupMapper)
	ghToGGMapping := make(GroupMapper)
	for _, v := range tm.GetMappings() {
		gitHubGroupID := github.Encode(v.GetGitHubTeam().GetOrgId(), v.GetGitHubTeam().GetTeamId())
		if _, ok := ggToGHMapping[v.GetGoogleGroup().GetGroupId()]; !ok {
			ggToGHMapping[v.GetGoogleGroup().GetGroupId()] = []string{gitHubGroupID}
		} else {
			ggToGHMapping[v.GetGoogleGroup().GetGroupId()] = append(ggToGHMapping[v.GetGoogleGroup().GetGroupId()], gitHubGroupID)
		}
		if _, ok := ghToGGMapping[gitHubGroupID]; !ok {
			ghToGGMapping[gitHubGroupID] = []string{v.GetGoogleGroup().GetGroupId()}
		} else {
			ghToGGMapping[gitHubGroupID] = append(ghToGGMapping[gitHubGroupID], v.GetGoogleGroup().GetGroupId())
		}
	}
	return ggToGHMapping, ghToGGMapping, nil
}

// NewOneToManyGroupMapper creates a groupsync.OneToManyMapper base on the input source
// and destination system type using provided groupMappingFile.
func NewBidirectionalNewOneToManyGroupMapper(source, dest, groupMappingFile string) (groupsync.OneToManyGroupMapper, groupsync.OneToManyGroupMapper, error) {
	if source == tltypes.SystemTypeGoogleGroups && dest == tltypes.SystemTypeGitHub {
		return NewBidirectionalGoogleGroupGitHubMapper(groupMappingFile)
	}
	return nil, nil, fmt.Errorf("unsupported source to dest mapper type: source %s, dest %s", source, dest)
}

// NewUserMapper creats a UserMapper base on source and dest system type.
func NewUserMapper(ctx context.Context, source, dest, mappingFilePath string) (groupsync.UserMapper, error) {
	if source == tltypes.SystemTypeGoogleGroups && dest == tltypes.SystemTypeGitHub {
		// return ggtogh.NewUserMapper(ctx, mappingFilePath)
		m, err := ggtogh.NewUserMapper(ctx, mappingFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create GoogleGroupGitHubUserMapper: %w", err)
		}
		return m, nil
	}
	return nil, fmt.Errorf("unsupported source to dest user mapper type: source %s, dest %s", source, dest)
}
