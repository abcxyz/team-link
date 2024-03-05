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

// SourceEventHandler interface that handles source event, typically it does:
//   1. Filter events
//     	- for teams and users that are in mapping.
//      - other conditions.
//   2. Check source and populate a snapshot of a GitHub team's memberships.
type SourceEventHandler interface {
  Handle(context.Context, *SourceEvent) (*GitHubTeam, error)
}

// Mapping interface that gets the mapped value given key value.
type Mapping interface {
  // Get the destination user mapped to the given source user.
  GetDestUser(ctx context.Context, srcUserEmail string) (string, error)

  // Get the destination team mapped to the given source team.
  GetDestTeam(ctx context.Context, srcTeam string)(string, error)

  // Get all source teams that are mapped to the same destination team.
  GetSourceTeams(ctx context.Context, destTeam string)([]string, error)
}
