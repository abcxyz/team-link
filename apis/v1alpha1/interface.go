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

package v1alpha1

import "context"

// GitHubTeamResolver interface that resolves source event and returns a list
// GitHubTeam objects.
type GitHubTeamResolver interface {
	// Resolve resolves a SourceEvent of a source team membership change and it
	// does the following:
	//  1. Find the GitHub teams that the source team is mapped to, and for each
	//     GitHub team, get all the source teams that it mapped to.
	//  2. Return a list of GitHubTeam objects and each contains all memberships
	//     of its mapped source teams and the GitHub team info, so that downstream
	//     can sync the memberships to the GitHub teams.
	Resolve(context.Context, *SourceEvent) ([]*GitHubTeam, error)
}

// GitHubMapper maps destination(GitHub) teams/users to source teams/users.
// The actual team and user ids depend on the type of source/destination systems.
type GitHubMapper interface {
	// GitHubUser returns a GitHubUser that contains the GitHub user
	// information for the given source user id.
	GitHubUser(ctx context.Context, srcUserID string) (*GitHubUser, error)

	// GitHubTeams returns a list of GitHubTeams that contains the GitHub team
	// information for the given source team id. Note that the relationship for
	// source to GitHub team could be many to many.
	GitHubTeams(ctx context.Context, srcTeamID string) ([]*GitHubTeam, error)
}

// TeamSynchronizer interface that syncs the team memberships in the each of the
// GitHubTeam from the given list to GitHub.
type TeamSynchronizer interface {
	// Sync syncs the team memberships in the each of the GitHubTeam from the
	// given list to GitHub.
	Sync(context.Context, []*GitHubTeam) error
}
