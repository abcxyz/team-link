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
	"github.com/abcxyz/team-link/pkg/github"
	"github.com/abcxyz/team-link/pkg/groupsync"
)

type GoogleGroupToGitHubMapper struct {
	GoogleGroupToGitHubTeam map[string][]string
}

// AllGroupIDs returns the set of groupIDs being mapped (the key set).
func (g *GoogleGroupToGitHubMapper) AllGroupIDs(ctx context.Context) ([]string, error) {
	res := make([]string, 0)
	for key := range g.GoogleGroupToGitHubTeam {
		res = append(res, key)
	}
	slices.Sort(res)
	return res, nil
}

// ContainsGroupID returns whether this mapper contains a mapping for the given group ID.
func (g *GoogleGroupToGitHubMapper) ContainsGroupID(ctx context.Context, groupID string) (bool, error) {
	_, ok := g.GoogleGroupToGitHubTeam[groupID]
	return ok, nil
}

// MappedGroupIDs returns the list of group IDs mapped to the given group ID.
func (g *GoogleGroupToGitHubMapper) MappedGroupIDs(ctx context.Context, groupID string) ([]string, error) {
	githubTeamIDs, ok := g.GoogleGroupToGitHubTeam[groupID]
	if !ok {
		return nil, fmt.Errorf("no mapping found for group ID: %s", groupID)
	}
	return githubTeamIDs, nil
}

// newGoogleGroupToGitHubMapper creates a GoogleGroupToGitHubMapper using the provided mapping file.
func newGoogleGroupToGitHubMapper(groupMappingFile string) (groupsync.OneToManyGroupMapper, error) {
	b, err := os.ReadFile(groupMappingFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping file: %w", err)
	}
	tm := &v1alpha3.GoogleGroupToGitHubTeamMappings{}
	if err := prototext.Unmarshal(b, tm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mapping file: %w", err)
	}
	mappings := make(map[string][]string)
	for _, v := range tm.GetMappings() {
		if _, ok := mappings[v.GetGoogleGroup().GetGroupId()]; !ok {
			mappings[v.GetGoogleGroup().GetGroupId()] = []string{github.Encode(v.GetGitHubTeam().GetOrgId(), v.GetGitHubTeam().GetTeamId())}
		} else {
			mappings[v.GetGoogleGroup().GetGroupId()] = append(mappings[v.GetGoogleGroup().GetGroupId()], github.Encode(v.GetGitHubTeam().GetOrgId(), v.GetGitHubTeam().GetTeamId()))
		}
	}
	return &GoogleGroupToGitHubMapper{
		GoogleGroupToGitHubTeam: mappings,
	}, nil
}

// NewOneToManyGroupMapper creates a groupsync.OneToManyMapper base on the input source
// and destination system type using provided groupMappingFile.
func NewOneToManyGroupMapper(source, dest, groupMappingFile string) (groupsync.OneToManyGroupMapper, error) {
	if source == tltypes.SystemTypeGoogleGroups && dest == tltypes.SystemTypeGitHub {
		return newGoogleGroupToGitHubMapper(groupMappingFile)
	}
	return nil, fmt.Errorf("unsupported source to dest mapper type: source %s, dest %s", source, dest)
}
