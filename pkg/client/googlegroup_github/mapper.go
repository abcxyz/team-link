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

// package googlegroupgithub provides mapping for GoogleGroup to GitHub.
package googlegroupgithub

import (
	"context"
	"fmt"
	"os"
	"strings"

	"google.golang.org/protobuf/encoding/prototext"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/team-link/apis/v1alpha3"
	"github.com/abcxyz/team-link/pkg/groupsync"
)

// GoogleGroupGitHubUserMapper implements groupsync.UserMapper.
type GoogleGroupGitHubUserMapper struct {
	mappings map[string]string
}

func (m *GoogleGroupGitHubUserMapper) MappedUserID(ctx context.Context, userID string) (string, error) {
	v, ok := m.mappings[userID]
	if !ok {
		return "", groupsync.ErrTargetUserIDNotFound
	}
	return v, nil
}

// NewUserMapper create a UserMapper for mapping from GoogleGroupUSer to GithubUser.
func NewUserMapper(ctx context.Context, userMappingFile string) (*GoogleGroupGitHubUserMapper, error) {
	logger := logging.FromContext(ctx)

	b, err := os.ReadFile(userMappingFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping file: %w", err)
	}
	var tm v1alpha3.UserMappings
	if err := prototext.Unmarshal(b, &tm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mapping file: %w", err)
	}
	ggToGHUserMapping := make(map[string]string)
	ghToGGUserMapping := make(map[string]string)

	for _, mapping := range tm.GetMappings() {
		src, dst := mapping.GetGoogleUserEmail(), mapping.GetGitHubUserId()
		// skip user if they don't have google group or github that needs mappings.
		if src == "" || dst == "" {
			continue
		}
		// Check user mapping relation is 1:1.
		if existingDst, ok := ggToGHUserMapping[src]; ok && existingDst != dst {
			logger.WarnContext(ctx, "duplicate github user mapped for same google group user",
				"google_group_user", src,
				"duplicaed_github_user", strings.Join([]string{existingDst, dst}, ","),
			)
		}
		ggToGHUserMapping[src] = dst

		if existingSrc, ok := ghToGGUserMapping[dst]; ok && existingSrc != src {
			logger.WarnContext(ctx, "duplicate google group user mapped for same github user",
				"github_user", dst,
				"duplicaed_github_user", strings.Join([]string{existingSrc, src}, ","),
			)
		}
		ghToGGUserMapping[dst] = src
	}
	return &GoogleGroupGitHubUserMapper{
		mappings: ggToGHUserMapping,
	}, nil
}
