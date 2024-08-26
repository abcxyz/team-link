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

package v1alpha2

import "context"

// SourceTeamClient handles getting information about Source Teams.
type SourceTeamClient interface {
	// Descendants retrieve all users (children, recursively) of a team.
	Descendants(ctx context.Context, sourceTeamID string) ([]string, error)
}

// GitHubMapper maps destination(GitHub) teams/users to source teams/users.
// The format of team and user IDs depend on the source team system.
type GitHubMapper interface {
	// GitHubUser returns a GitHubUser that contains the GitHub user
	// information for the given source user id.
	GitHubUser(ctx context.Context, sourceUserID string) (*GitHubUser, error)

	// GitHubTeams returns a list of GitHubTeams that contains the GitHub team
	// information for the given source team id. Note that the relationship for
	// source to GitHub team could be many to many.
	GitHubTeams(ctx context.Context, sourceTeamID string) ([]*GitHubTeam, error)

	// ContainsMappingForTeamID returns whether this mapper has an entry for the given source team ID.
	ContainsMappingForTeamID(ctx context.Context, sourceTeamID string) bool

	// SourceTeamIDs returns the list of source team IDs for which this mapper has mappings.
	SourceTeamIDs(ctx context.Context) []string
}

// TeamSynchronizer interface that syncs the team memberships for each of given GitHubTeams to GitHub.
type TeamSynchronizer interface {
	// Sync syncs the team memberships for each of given GitHubTeams to GitHub.
	Sync(context.Context, []*GitHubTeam) error
}
