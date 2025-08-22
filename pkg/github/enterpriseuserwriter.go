// Copyright 2025 The Authors (see AUTHORS file)
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

package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-github/v67/github"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/team-link/pkg/groupsync"
)

const defaultMaxUsersToProvision = 1000

var _ groupsync.GroupWriter = (*EnterpriseUserWriter)(nil)

// ENterpriseRWOpt is a configuration option for EnterpriseUserReadWriter.
type EnterpriseRWOpt func(rw *EnterpriseUserWriter)

// WithMaxUsersToProvision sets the maximum number of SCIM provisioned users.
func WithMaxUsersToProvision(num int64) EnterpriseRWOpt {
	return func(rw *EnterpriseUserWriter) {
		rw.maxUsersToProvision = num
	}
}

// EnterpriseUserWriter manages enterprise users via a direct GHES SCIM API client.
type EnterpriseUserWriter struct {
	scimClient          *SCIMClient
	maxUsersToProvision int64
}

// NewEnterpriseUserWriter creates a new EnterpriseUserWriter with default 1000
// maximum number of users to provision if not override by given opts.
func NewEnterpriseUserWriter(httpClient *http.Client, enterpriseBaseURL string, opts ...EnterpriseRWOpt) (*EnterpriseUserWriter, error) {
	scimClient, err := NewSCIMClient(httpClient, enterpriseBaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create scim client: %w", err)
	}
	w := &EnterpriseUserWriter{
		maxUsersToProvision: defaultMaxUsersToProvision,
		scimClient:          scimClient,
	}
	for _, opt := range opts {
		opt(w)
	}
	return w, nil
}

// SetMembers creates and suspends enterprise users given the desired members.
func (w *EnterpriseUserWriter) SetMembers(ctx context.Context, _ string, members []groupsync.Member) error {
	logger := logging.FromContext(ctx)

	currentUsersMap, err := w.scimClient.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}
	desiredUsersMap := make(map[string]*github.SCIMUserAttributes)
	desiredUsersName := []string{}
	for _, m := range members {
		if !m.IsUser() {
			logger.DebugContext(ctx, "skipping non-user member", "member", m.ID())
			continue
		}
		u, _ := m.User()
		scimUser, ok := u.Attributes.(*github.SCIMUserAttributes)
		if !ok {
			logger.DebugContext(ctx, "skipping non-SCIM user member", "member", m.ID())
			continue
		}
		desiredUsersMap[scimUser.UserName] = scimUser
		desiredUsersName = append(desiredUsersName, scimUser.UserName)
	}

	var merr error
	// 1. Deactivate users to free up license seats first.
	for username, scimUser := range currentUsersMap {
		// Skip deactivated user
		if scimUser.Active != nil && !*scimUser.Active {
			continue
		}
		// Deactivate user who is not in desiredUsersMap
		if _, ok := desiredUsersMap[username]; !ok {
			logger.InfoContext(ctx, "deactivating user", "user", username)
			if _, _, err := w.scimClient.DeactivateUser(ctx, *scimUser.ID); err != nil {
				merr = errors.Join(merr, fmt.Errorf("failed to deactivate %q: %w", username, err))
			}
		}
	}

	// 2. Create and reactivate users.
	var count int64
	for _, username := range desiredUsersName {
		count++
		if count > w.maxUsersToProvision {
			merr = errors.Join(merr, fmt.Errorf("exceeded max users to provision: %d", w.maxUsersToProvision))
			break
		}

		scimUser, ok := currentUsersMap[username]
		if !ok {
			// Create user if not found in currentUsersMap
			logger.InfoContext(ctx, "creating user", "user", username)
			if _, _, err := w.scimClient.CreateUser(ctx, desiredUsersMap[username]); err != nil {
				merr = errors.Join(merr, fmt.Errorf("failed to create %q: %w", username, err))
			}
		} else {
			// Reactivate user if user status is unknown or deactivated.
			if scimUser.Active == nil || !*scimUser.Active {
				logger.InfoContext(ctx, "reactivating user", "user", username)
				if _, _, err := w.scimClient.ReactivateUser(ctx, *scimUser.ID); err != nil {
					merr = errors.Join(merr, fmt.Errorf("failed to reactivate %q: %w", username, err))
				}
			}
		}
	}
	return merr
}
