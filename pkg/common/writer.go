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
	"github.com/abcxyz/team-link/v2/pkg/github"
	"github.com/abcxyz/team-link/v2/pkg/groupsync"
)

// NewReadWriter creates a new ReadWriter base on target system type and provided config.
func NewReadWriter(ctx context.Context, target string, config *api.TeamLinkConfig, mappings *api.TeamLinkMappings) (groupsync.GroupReadWriter, error) {
	if target == tltypes.SystemTypeGitHub {
		readWriter, err := NewGitHubReadWriter(ctx, config.GetTargetConfig().GetGithubConfig(), mappings)
		if err != nil {
			return nil, fmt.Errorf("failed to create readwriter for github: %w", err)
		}
		return readWriter, nil
	}
	return nil, fmt.Errorf("unsupported system type %s", target)
}

// NewGitHubReadWriter creates a ReadWriter for github using provided config.
func NewGitHubReadWriter(ctx context.Context, config *api.GitHubConfig, mappings *api.TeamLinkMappings) (groupsync.GroupReadWriter, error) {
	orgTeamSSORequired := computeOrgTeamSSORequired(mappings)
	switch a := config.GetAuthentication().(type) {
	case *api.GitHubConfig_StaticAuth:
		tokenSource, err := github.NewStaticTokenSourceFromEnvVar(a.StaticAuth.GetFromEnvironment())
		if err != nil {
			return nil, fmt.Errorf("failed to create StaticTokenSource: %w", err)
		}
		writer, err := github.NewTeamReadWriterWithStaticTokenSource(ctx, tokenSource, config.GetEnterpriseUrl(), orgTeamSSORequired)
		if err != nil {
			return nil, fmt.Errorf("failed to create readwriter: %w", err)
		}
		return writer, nil
	}
	return nil, fmt.Errorf("unsupported authentication type method for github")
}

// computeOrgTeamSSORequired compute whether a team in a org requires
// user to have SSO enabled to do membership syncing using the provided
// api.TeamLinkMappings. The result is stored as a map of type
// map[int64]map[int64]bool.
//
// For example:
// If team `abc` under org `xyz` required users to have SSO enabled,
// we will have orgTeamSSORequired["xyz"]["abc"] = true
// If team `foo` under org `bar` doesn't require user to have SSO enabled,
// we will have orgTeamSSORequired["bar"]["foo"] = false.
func computeOrgTeamSSORequired(mappings *api.TeamLinkMappings) map[int64]map[int64]bool {
	orgTeamSSORequired := make(map[int64]map[int64]bool)
	for _, v := range mappings.GetGroupMappings().GetMappings() {
		if _, ok := orgTeamSSORequired[v.GetGithub().GetOrgId()]; !ok {
			orgTeamSSORequired[v.GetGithub().GetOrgId()] = make(map[int64]bool)
			orgTeamSSORequired[v.GetGithub().GetOrgId()][v.GetGithub().GetTeamId()] = v.GetGithub().GetRequireUserEnableSso()
		} else {
			orgTeamSSORequired[v.GetGithub().GetOrgId()][v.GetGithub().GetTeamId()] = v.GetGithub().GetRequireUserEnableSso()
		}
	}
	return orgTeamSSORequired
}
