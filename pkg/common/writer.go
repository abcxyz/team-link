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

	api "github.com/abcxyz/team-link/apis/v1alpha3/proto"
	tltypes "github.com/abcxyz/team-link/internal"
	"github.com/abcxyz/team-link/pkg/github"
	"github.com/abcxyz/team-link/pkg/groupsync"
)

// NewReadWriter creates a new ReadWriter base on target system type and provided config.
func NewReadWriter(ctx context.Context, target string, config *api.TeamLinkConfig) (groupsync.GroupReadWriter, error) {
	if target == tltypes.SystemTypeGitHub {
		readWriter, err := NewGitHubReadWriter(ctx, config.GetTargetConfig().GetGithubConfig())
		if err != nil {
			return nil, fmt.Errorf("failed to create readwriter for github: %w", err)
		}
		return readWriter, nil
	}
	return nil, fmt.Errorf("unsupported system type %s", target)
}

// NewGitHubReadWriter creates a ReadWriter for github using provided config.
func NewGitHubReadWriter(ctx context.Context, config *api.GitHubConfig) (groupsync.GroupReadWriter, error) {
	switch a := config.GetAuthentication().(type) {
	case *api.GitHubConfig_StaticAuth:
		tokenSource, err := github.NewStaticTokenSourceFromEnvVar(a.StaticAuth.GetFromEnvironment())
		if err != nil {
			return nil, fmt.Errorf("failed to create StaticTokenSource: %w", err)
		}
		writer, err := github.NewTeamReadWriterWithStaticTokenSource(ctx, tokenSource, config.GetEnterpriseUrl())
		if err != nil {
			return nil, fmt.Errorf("failed to create readwriter: %w", err)
		}
		return writer, nil
	}
	return nil, fmt.Errorf("unsupported authentication type method for github")
}
