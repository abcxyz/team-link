// Copyright 2024 Google LLC
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

// package googlegroupgithub provides mapping for GoogleGroup to GitHub.
package googlegroupgithub

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/abcxyz/pkg/logging"
	api "github.com/abcxyz/team-link/v2/apis/v1alpha3/proto"
	"github.com/abcxyz/team-link/v2/pkg/github"
	"github.com/abcxyz/team-link/v2/pkg/groupsync"
)

// GroupMapper implements groupsync.OneToManyGroupMapper
// For now it was be used as Mapper between GoogleGroup
// and GitHubTeam.
// Both GoogleGroupMapper and GitHubMapper have similar
// variables and functions, this way we don't need to repeatly
// write similar function for both on them.
type GroupMapper struct {
	mappings map[string][]string
}

func (m *GroupMapper) AllGroupIDs(ctx context.Context) ([]string, error) {
	res := make([]string, 0, len(m.mappings))
	for key := range m.mappings {
		res = append(res, key)
	}
	slices.Sort(res)
	return res, nil
}

func (m *GroupMapper) ContainsGroupID(ctx context.Context, key string) (bool, error) {
	_, ok := m.mappings[key]
	return ok, nil
}

func (m *GroupMapper) MappedGroupIDs(ctx context.Context, key string) ([]string, error) {
	x, ok := m.mappings[key]
	if !ok {
		return nil, fmt.Errorf("no mapping found for group ID: %s", key)
	}
	// Make deep copy so the caller's operation on return won't change
	// the value of this given map.
	ret := make([]string, len(x))
	copy(ret, x)
	return ret, nil
}

func (m *GroupMapper) Mappings(ctx context.Context, key string) ([]groupsync.Mapping, error) {
	mappedGroupIDs, err := m.MappedGroupIDs(ctx, key)
	if err != nil {
		return nil, err
	}
	mappings := make([]groupsync.Mapping, len(mappedGroupIDs))
	for i, groupID := range mappedGroupIDs {
		mappings[i] = groupsync.Mapping{
			GroupID: groupID,
		}
	}
	return mappings, nil
}

type BiDirectionalGroupMapper struct {
	SourceMapper *GroupMapper
	TargetMapper *GroupMapper
}

func NewBidirectionalGroupMapper(mappings *api.GroupMappings) *BiDirectionalGroupMapper {
	ggToGHMapping := make(map[string][]string)
	ghToGGMapping := make(map[string][]string)
	for _, v := range mappings.GetMappings() {
		gitHubGroupID := github.Encode(v.GetGithub().GetOrgId(), v.GetGithub().GetTeamId())
		ggGroupID := v.GetGoogleGroups().GetGroupId()
		ggToGHMapping[ggGroupID] = append(ggToGHMapping[ggGroupID], gitHubGroupID)
		ghToGGMapping[gitHubGroupID] = append(ghToGGMapping[gitHubGroupID], ggGroupID)
	}
	return &BiDirectionalGroupMapper{
		SourceMapper: &GroupMapper{mappings: ggToGHMapping},
		TargetMapper: &GroupMapper{mappings: ghToGGMapping},
	}
}

// GoogleGroupGitHubUserMapper implements groupsync.UserMapper.
type GoogleGroupGitHubUserMapper struct {
	mappings map[string]string
}

func (m *GoogleGroupGitHubUserMapper) MappedUserID(ctx context.Context, userID string) (string, error) {
	v, ok := m.mappings[userID]
	if !ok {
		return "", groupsync.ErrTargetUserIDNotFound
	}
	return v, nil
}

func (m *GoogleGroupGitHubUserMapper) MappedUser(ctx context.Context, user *groupsync.User) (*groupsync.User, error) {
	v, ok := m.mappings[user.ID]
	if !ok {
		return nil, groupsync.ErrTargetUserIDNotFound
	}
	return &groupsync.User{
		ID:       v,
		Metadata: user.Metadata,
	}, nil
}

// NewUserMapper create a UserMapper for mapping from GoogleGroupUSer to GithubUser.
func NewUserMapper(ctx context.Context, mappings *api.UserMappings) *GoogleGroupGitHubUserMapper {
	logger := logging.FromContext(ctx)

	ggToGHUserMapping := make(map[string]string)
	ghToGGUserMapping := make(map[string]string)

	for _, mapping := range mappings.GetMappings() {
		src, dst := mapping.GetSource(), mapping.GetTarget()
		// skip user if they don't have google group or github that needs mappings.
		if src == "" || dst == "" {
			continue
		}
		// Check user mapping relation is 1:1.
		if existingDst, ok := ggToGHUserMapping[src]; ok && existingDst != dst {
			logger.WarnContext(ctx, "duplicate github user mapped for same google group user",
				"google_group_user", src,
				"duplicaed_github_user", strings.Join([]string{existingDst, dst}, ","),
			)
		}
		ggToGHUserMapping[src] = dst

		if existingSrc, ok := ghToGGUserMapping[dst]; ok && existingSrc != src {
			logger.WarnContext(ctx, "duplicate google group user mapped for same github user",
				"github_user", dst,
				"duplicaed_github_user", strings.Join([]string{existingSrc, src}, ","),
			)
		}
		ghToGGUserMapping[dst] = src
	}
	return &GoogleGroupGitHubUserMapper{
		mappings: ggToGHUserMapping,
	}
}
