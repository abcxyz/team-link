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

	tltypes "github.com/abcxyz/team-link/internal"
	tlgithub "github.com/abcxyz/team-link/pkg/github"
	"github.com/abcxyz/team-link/pkg/groupsync"
)

// NewReadWriter creates a GroupReadWriter base on provided destination type.
func NewReadWrirter(ctx context.Context, destination, token string) (groupsync.GroupReadWriter, error) {
	if destination == tltypes.SystemTypeGitHub {
		readWriter := tlgithub.NewGitHubTeamReadWriterWithAuthToken(ctx, token)
		return readWriter, nil
	}
	return nil, fmt.Errorf("destination type %s not allowed", destination)
}
