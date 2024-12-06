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

// package googlegroupgithub provides mapping for GoogleGroup to GitHub.
package googlegroupgithub

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"google.golang.org/protobuf/encoding/prototext"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/team-link/apis/v1alpha3"
	"github.com/abcxyz/team-link/pkg/github"
	"github.com/abcxyz/team-link/pkg/groupsync"
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

type BiDirectionalGroupMapper struct {
	sourceMapper *GroupMapper
	targetMapper *GroupMapper
}

// NewBidirectionalGoogleGroupGitHubMapper creates a GoogleGroupToGitHubMapper
// and a GitHubToGoogleGroupMapper using the provided groupMapping file.
// Returns is (GoogleGroupToGitHubMapper, GitHubToGoogleGroupMapper, error).
//
// TODO: refactor this into client/googlegroup_github/mapper.go later.
func NewBidirectionaGroupMapper(groupMappingFile string) (*BiDirectionalGroupMapper, error) {
	b, err := os.ReadFile(groupMappingFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping file: %w", err)
	}
	tm := &v1alpha3.GoogleGroupToGitHubTeamMappings{}
	if err := prototext.Unmarshal(b, tm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mapping file: %w", err)
	}
	ggToGHMapping := make(map[string][]string)
	ghToGGMapping := make(map[string][]string)
	for _, v := range tm.GetMappings() {
		gitHubGroupID := github.Encode(v.GetGitHubTeam().GetOrgId(), v.GetGitHubTeam().GetTeamId())
		ggToGHMapping[v.GetGoogleGroup().GetGroupId()] = append(ggToGHMapping[v.GetGoogleGroup().GetGroupId()], gitHubGroupID)
		ghToGGMapping[gitHubGroupID] = append(ghToGGMapping[gitHubGroupID], v.GetGoogleGroup().GetGroupId())
	}
	return &BiDirectionalGroupMapper{
		sourceMapper: &GroupMapper{mappings: ggToGHMapping},
		targetMapper: &GroupMapper{mappings: ghToGGMapping},
	}, nil
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

// NewUserMapper create a UserMapper for mapping from GoogleGroupUSer to GithubUser.
func NewUserMapper(ctx context.Context, userMappingFile string) (*GoogleGroupGitHubUserMapper, error) {
	logger := logging.FromContext(ctx)

	b, err := os.ReadFile(userMappingFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping file: %w", err)
	}
	var tm v1alpha3.UserMappings
	if err := prototext.Unmarshal(b, &tm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mapping file: %w", err)
	}
	ggToGHUserMapping := make(map[string]string)
	ghToGGUserMapping := make(map[string]string)

	for _, mapping := range tm.GetMappings() {
		src, dst := mapping.GetGoogleUserEmail(), mapping.GetGitHubUserId()
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
	}, nil
}
