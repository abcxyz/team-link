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

// package github defines the mechanism to update GitHub teams' memberships
// given a list of records of the expected team memberships.
package github

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/google/go-github/v61/github"

	"github.com/abcxyz/pkg/githubauth"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/sets"
	"github.com/abcxyz/team-link/apis/v1alpha1"
)

// Synchronizer that syncs github team memberships.
// TODO(#7): add retry strategy.
type Synchronizer struct {
	client    *github.Client
	githubApp *githubauth.App
	dryrun    bool
}

// Option to set up a synchronizer.
type Option func(*Synchronizer) error

// Enable dryrun mode for synchronizer, in dryrun mode, the synchronizer will
// only log the membership updates instead of doing a real sync.
func WithDryRun() Option {
	return func(s *Synchronizer) error {
		s.dryrun = true
		return nil
	}
}

// NewSynchronizer creates a new Synchronizer with provided clients and options.
func NewSynchronizer(ghClient *github.Client, ghApp *githubauth.App, opts ...Option) (*Synchronizer, error) {
	s := &Synchronizer{
		client:    ghClient,
		githubApp: ghApp,
	}
	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, fmt.Errorf("failed to set option: %w", err)
		}
	}
	return s, nil
}

// Sync overides several GitHub teams' memberships with the provided team
// membership snapshots.
// TODO(#3): populate the users' GitHub logins in the GitHubTeam object before
// this since they are required when updating GitHub team memberships.
func (s *Synchronizer) Sync(ctx context.Context, teams []*v1alpha1.GitHubTeam) error {
	logger := logging.FromContext(ctx)

	var retErr error
	for _, team := range teams {
		// Get the GitHub client that uses the auth token from the right org
		// installation.
		ghClient, err := s.githubClient(ctx, team.GetOrgId())
		if err != nil {
			retErr = errors.Join(retErr, err)
			continue
		}

		// Get current team members' login from GitHub and expected team members'
		// user from the team object.
		gotLogins, err := s.currentTeamLogins(ctx, ghClient, team.GetOrgId(), team.GetTeamId())
		if err != nil {
			retErr = errors.Join(
				retErr,
				fmt.Errorf("failed to get GitHub team members/invitations for team(%d): %w", team.GetTeamId(), err),
			)
			continue
		}
		wantLogins := loginsFromTeam(team)

		add := sets.Subtract(wantLogins, gotLogins)
		remove := sets.Subtract(gotLogins, wantLogins)
		if s.dryrun {
			logger.InfoContext(
				ctx,
				"dryrun mode is on, skip updating memberships",
				"team", team.GetTeamId(),
				"users_to_add", add,
				"users_to_remove", remove,
			)
			continue
		}

		// Add GitHub team memberships.
		for _, u := range add {
			if _, _, err := ghClient.Teams.AddTeamMembershipByID(
				ctx,
				team.GetOrgId(),
				team.GetTeamId(),
				u,
				&github.TeamAddTeamMembershipOptions{Role: "member"},
			); err != nil {
				retErr = errors.Join(
					retErr,
					fmt.Errorf("failed to add GitHub team members for team(%d): %w", team.GetTeamId(), err),
				)
			}
		}
		// Remove GitHub team memberships
		for _, u := range remove {
			// If it is a pending invitation, RemoveTeamMembershipByID will cancel the
			// pending invitation for the team and for that user.
			if _, err := ghClient.Teams.RemoveTeamMembershipByID(ctx, team.GetOrgId(), team.GetTeamId(), u); err != nil {
				retErr = errors.Join(
					retErr,
					fmt.Errorf("failed to remove GitHub team members for team(%d): %w", team.GetTeamId(), err),
				)
			}
		}
	}
	return retErr
}

// currentTeamLogins returns a list of GitHub logins that are members or has
// invitations to the given GitHub team.
// TODO(#6): make the paginated GitHub API call generic.
func (s *Synchronizer) currentTeamLogins(ctx context.Context, c *github.Client, orgID, teamID int64) ([]string, error) {
	loginsMap := make(map[string]struct{}, 32)

	if err := paginate(func(listOpts *github.ListOptions) (*github.Response, error) {
		opts := &github.TeamListTeamMembersOptions{
			Role:        "all",
			ListOptions: *listOpts,
		}

		members, resp, err := c.Teams.ListTeamMembersByID(ctx, orgID, teamID, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list team membership: %w", err)
		}

		for _, m := range members {
			// just checking, login should be provided for active members.
			if v := m.GetLogin(); v != "" {
				loginsMap[v] = struct{}{}
			}
		}
		return resp, nil
	}); err != nil {
		return nil, err
	}

	if err := paginate(func(listOpts *github.ListOptions) (*github.Response, error) {
		invitations, resp, err := c.Teams.ListPendingTeamInvitationsByID(ctx, orgID, teamID, listOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list team invitations: %w", err)
		}

		for _, inv := range invitations {
			// login could be missing if the invitation is sent to an email.
			if v := inv.GetLogin(); v != "" {
				loginsMap[v] = struct{}{}
			}
		}
		return resp, nil
	}); err != nil {
		return nil, err
	}

	// convert logins to a slice and sort
	logins := make([]string, 0, len(loginsMap))
	for k := range loginsMap {
		logins = append(logins, k)
	}
	sort.Strings(logins)

	return logins, nil
}

// githubClient returns a github client configured to use an installation access
// token.
func (s *Synchronizer) githubClient(ctx context.Context, orgID int64) (*github.Client, error) {
	tr := &githubauth.TokenRequestAllRepos{
		Permissions: map[string]string{
			"members": "write",
		},
	}
	appInstallation, err := s.githubApp.InstallationForOrg(ctx, strconv.FormatInt(orgID, 10))
	if err != nil {
		return nil, fmt.Errorf("failed get GitHub App installation: %w", err)
	}
	token, err := appInstallation.AccessTokenAllRepos(ctx, tr)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub App installtion access token: %w", err)
	}
	return s.client.WithAuthToken(token), nil
}

// loginsFromTeam returns a list of GitHub logins/usernames that are in the
// given team object.
func loginsFromTeam(team *v1alpha1.GitHubTeam) []string {
	res := make([]string, len(team.GetUsers()))
	for i, m := range team.GetUsers() {
		res[i] = m.GetLogin()
	}
	return res
}
