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

	"github.com/google/go-github/v61/github"

	"github.com/abcxyz/pkg/githubauth"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/team-link/apis/v1alpha1"
)

// Synchronizer that syncs github team memberships.
// TODO(#7): add retry strategy.
type Synchronizer struct {
	client    *github.Client
	githubApp *githubauth.App
}

// NewSynchronizer creates a new Synchronizer with provided clients and options.
func NewSynchronizer(ghClient *github.Client, ghApp *githubauth.App) *Synchronizer {
	return &Synchronizer{
		client:    ghClient,
		githubApp: ghApp,
	}
}

// Sync overrides several GitHub teams' memberships with the provided team
// membership snapshots.
// TODO(#3): populate the users' GitHub logins in the GitHubTeam object before
// this since they are required when updating GitHub team memberships.
func (s *Synchronizer) Sync(ctx context.Context, teams []*v1alpha1.GitHubTeam) error {
	logger := logging.FromContext(ctx)

	// Configure Github auth token to the GitHub client.
	t, err := s.accessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}
	ghClient := s.client.WithAuthToken(t)

	var retErr error
	for _, team := range teams {
		// Get current team members including both active member and pending
		// membership invitations from GitHub.
		gotMembers, err := s.currentTeamMembers(ctx, ghClient, team.GetOrgId(), team.GetTeamId())
		if err != nil {
			retErr = errors.Join(
				retErr,
				fmt.Errorf("failed to get GitHub team members/invitations for team(%d): %w", team.GetTeamId(), err),
			)
			continue
		}

		// Add GitHub team memberships.
		gotMemberLogins := loginsFromUsers(gotMembers)
		gotMemberEmails := emailsFromUsers(gotMembers)
		for _, u := range team.GetUsers() {
			if _, ok := gotMemberLogins[u.GetLogin()]; ok {
				continue
			}
			if _, ok := gotMemberEmails[u.GetEmail()]; ok {
				continue
			}
			if u.GetLogin() == "" {
				logger.WarnContext(
					ctx,
					"skip adding membership by user email, please provide user login instead",
					"user_email", u.GetEmail())
				continue
			}
			if _, _, err := ghClient.Teams.AddTeamMembershipByID(
				ctx,
				team.GetOrgId(),
				team.GetTeamId(),
				u.GetLogin(),
				&github.TeamAddTeamMembershipOptions{Role: "member"},
			); err != nil {
				retErr = errors.Join(
					retErr,
					fmt.Errorf("failed to add GitHub team members for team(%d): %w", team.GetTeamId(), err),
				)
			}
		}

		// Remove GitHub team memberships.
		wantMemberLogins := loginsFromUsers(team.GetUsers())
		wantMemberEmails := emailsFromUsers(team.GetUsers())
		for _, u := range gotMembers {
			if _, ok := wantMemberLogins[u.GetLogin()]; ok {
				continue
			}
			if _, ok := wantMemberEmails[u.GetEmail()]; ok {
				continue
			}
			if u.GetLogin() == "" {
				// We don't cancel pending invitations since one invitation could
				// involve multiple teams, see team_ids parameter in GitHub API
				// https://docs.github.com/en/rest/orgs/members?apiVersion=2022-11-28#create-an-organization-invitation
				logger.InfoContext(
					ctx,
					"skip removing membership by user email, it is probably a pending invitation",
					"user_email", u.GetEmail())
				continue
			}
			if _, err := ghClient.Teams.RemoveTeamMembershipByID(ctx, team.GetOrgId(), team.GetTeamId(), u.GetLogin()); err != nil {
				retErr = errors.Join(
					retErr,
					fmt.Errorf("failed to remove GitHub team members for team(%d): %w", team.GetTeamId(), err),
				)
			}
		}
	}
	return retErr
}

// currentTeamMembers returns a list of GitHub users that are members or has
// invitations to the given GitHub team.
// TODO(#6): make the paginated GitHub API call generic.
func (s *Synchronizer) currentTeamMembers(
	ctx context.Context,
	c *github.Client,
	orgID, teamID int64,
) ([]*v1alpha1.GitHubUser, error) {
	callMap := []func(
		ctx context.Context,
		c *github.Client,
		orgID, teamID int64,
		opt *github.ListOptions,
	) ([]*v1alpha1.GitHubUser, *github.Response, error){
		listActiveTeamMembers,
		listPendingTeamInvitations,
	}
	var res []*v1alpha1.GitHubUser
	for _, f := range callMap {
		opt := &github.ListOptions{
			PerPage: 100,
		}
		for {
			users, resp, err := f(ctx, c, orgID, teamID, opt)
			if err != nil {
				return nil, fmt.Errorf("failed to get team member/invitation logins: %w", err)
			}

			res = append(res, users...)
			if resp.NextPage == 0 {
				break // No more pages to fetch
			}
			opt.Page = resp.NextPage
		}
	}
	return res, nil
}

func listActiveTeamMembers(
	ctx context.Context,
	c *github.Client,
	orgID, teamID int64,
	opt *github.ListOptions,
) ([]*v1alpha1.GitHubUser, *github.Response, error) {
	o := &github.TeamListTeamMembersOptions{
		Role:        "all",
		ListOptions: *opt,
	}
	members, resp, err := c.Teams.ListTeamMembersByID(ctx, orgID, teamID, o)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list team membership: %w", err)
	}
	users := make([]*v1alpha1.GitHubUser, 0, len(members))
	for _, m := range members {
		u := &v1alpha1.GitHubUser{}
		if m.Login != nil {
			u.Login = *m.Login
		}
		if m.Email != nil {
			u.Email = *m.Email
		}
		users = append(users, u)
	}
	return users, resp, nil
}

func listPendingTeamInvitations(
	ctx context.Context,
	c *github.Client,
	orgID, teamID int64,
	opt *github.ListOptions,
) ([]*v1alpha1.GitHubUser, *github.Response, error) {
	invitations, resp, err := c.Teams.ListPendingTeamInvitationsByID(ctx, orgID, teamID, opt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list team invitations: %w", err)
	}
	users := make([]*v1alpha1.GitHubUser, 0, len(invitations))
	for _, inv := range invitations {
		u := &v1alpha1.GitHubUser{}
		// It is possible that login is missing when an invitation is sent to an
		// email.
		if inv.Login != nil {
			u.Login = *inv.Login
		}
		if inv.Email != nil {
			u.Email = *inv.Email
		}
		users = append(users, u)
	}
	return users, resp, nil
}

func (s *Synchronizer) accessToken(ctx context.Context) (string, error) {
	tr := &githubauth.TokenRequestAllRepos{
		Permissions: map[string]string{
			"members": "write",
		},
	}

	token, err := s.githubApp.AccessTokenAllRepos(ctx, tr)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	return token, nil
}

// loginsFromUsers returns a set/map of GitHub logins/usernames that are in the
// given user list.
func loginsFromUsers(us []*v1alpha1.GitHubUser) map[string]struct{} {
	res := make(map[string]struct{})
	for _, m := range us {
		if m.GetLogin() != "" {
			res[m.GetLogin()] = struct{}{}
		}
	}
	return res
}

// emailsFromUsers returns a set/map of GitHub user email that are in the
// given user list.
func emailsFromUsers(us []*v1alpha1.GitHubUser) map[string]struct{} {
	res := make(map[string]struct{})
	for _, m := range us {
		if m.GetEmail() != "" {
			res[m.GetEmail()] = struct{}{}
		}
	}
	return res
}
