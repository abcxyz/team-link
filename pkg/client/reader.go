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
	"github.com/abcxyz/team-link/pkg/googlegroups"
	"github.com/abcxyz/team-link/pkg/groupsync"
)

// NewReader creates a GroupReader base on provided source type.
func NewReader(ctx context.Context, source string) (groupsync.GroupReader, error) {
	if source == tltypes.SystemTypeGoogleGroups {
		r, err := googlegroups.NewReader(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create reader for %s: %w", source, err)
		}
		return r, nil
	}
	return nil, fmt.Errorf("source type %s not allowd", source)
}
