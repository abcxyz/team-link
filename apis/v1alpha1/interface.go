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

// Mapper interface that gets the mapped value given key value.
type Mapper interface {
  // Get the destination user id mapped to the given source user id.
  DestUserId(ctx context.Context, srcUserId string) (string, error)

  // Get the destination team id mapped to the given source team id.
  DestTeamId(ctx context.Context, srcTeamId string)(string, error)

  // Get all source teams's ids that are mapped to the same destination team id.
  SourceTeamIds(ctx context.Context, destTeamId string)([]string, error)
}
