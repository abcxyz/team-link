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

// SourceEventHandler interface that handles source event.
type SourceEventHandler interface {
	// Handle handles a SourceEvent of a source team membership change and it
	// typically does the following:
	//   1. Find the GitHub team that the source team is mapped to, and get all
	//      the source teams that are mapped to the same GitHub team.
	//   2. Return a GitHubTeam object that contains all memberships of these
	//      source teams and the GitHub team info, so that downstream can
	//      sync the memberships to the Github team.
	Handle(context.Context, *SourceEvent) (*GitHubTeam, error)
}

// Mapper maps source teams/users and destination teams/users.
// The actual team and user ids depend on the type of source/destination systems.
type Mapper interface {
	// DestUserID returns the destination user id for the given source user id.
	DestUserID(ctx context.Context, srcUserID string) (string, error)

	// DestTeamID returns the destination team id for the given source team id.
	DestTeamID(ctx context.Context, srcTeamID string) (string, error)

	// SourceTeamIDs returns all source teams' ids for the given destination team
	// id. One destination team could have multiple source teams.
	SourceTeamIDs(ctx context.Context, destTeamID string) ([]string, error)
}
