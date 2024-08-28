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

// Package teamlink defines a general purpose TeamLinkService.
package teamlink

import (
	"context"
	"errors"
	"fmt"

	"github.com/abcxyz/team-link/apis/v1alpha2"
)

type SyncerFunc func(ctx context.Context) (v1alpha2.TeamSynchronizer, error)

type TeamLinkService struct {
	sourceTeamClient v1alpha2.SourceTeamClient
	githubMapper     v1alpha2.GitHubMapper
	syncerFunc       SyncerFunc
}

func New(sourceTeamClient v1alpha2.SourceTeamClient, githubMapper v1alpha2.GitHubMapper, syncerFunc SyncerFunc) *TeamLinkService {
	return &TeamLinkService{
		sourceTeamClient: sourceTeamClient,
		githubMapper:     githubMapper,
		syncerFunc:       syncerFunc,
	}
}

// SyncTeam syncs a single team to GitHub.
func (t *TeamLinkService) SyncTeam(ctx context.Context, srcEvent *v1alpha2.SourceEvent) error {
	syncer, err := t.syncerFunc(ctx)
	if err != nil {
		return fmt.Errorf("failed to get syncer: %w", err)
	}
	return t.syncTeam(ctx, srcEvent, syncer)
}

// SyncAllTeams syncs all teams to GitHub.
func (t *TeamLinkService) SyncAllTeams(ctx context.Context) error {
	syncer, err := t.syncerFunc(ctx)
	if err != nil {
		return fmt.Errorf("failed to get syncer: %w", err)
	}
	var merr error
	for _, srcTeamID := range t.githubMapper.SourceTeamIDs(ctx) {
		if err := t.syncTeam(ctx, &v1alpha2.SourceEvent{TeamId: srcTeamID}, syncer); err != nil {
			merr = errors.Join(merr, fmt.Errorf("failed to sync source team ID %s: %w", srcTeamID, err))
		}
	}
	return merr
}

func (t *TeamLinkService) syncTeam(ctx context.Context, srcEvent *v1alpha2.SourceEvent, syncer v1alpha2.TeamSynchronizer) error {
	// resolve might return a list of teams and an error. The error is non-nil if something went
	// wrong with some of the teams. In this case, we still want to sync the teams that were resolved
	// successfully.
	githubTeams, merr := t.resolve(ctx, srcEvent)
	if len(githubTeams) == 0 {
		return merr
	}
	if err := syncer.Sync(ctx, githubTeams); err != nil {
		merr = errors.Join(merr, fmt.Errorf("failed to sync some/all teams: %w", err))
	}
	return merr
}

func (t *TeamLinkService) resolve(ctx context.Context, event *v1alpha2.SourceEvent) ([]*v1alpha2.GitHubTeam, error) {
	// Get dest teams by source group.
	teams, err := t.githubMapper.GitHubTeams(ctx, event.GetTeamId())
	if len(teams) == 0 {
		return nil, err //nolint:wrapcheck // Want passthrough
	}
	retErr := err

	// For each of the team, get its descendants from source.
	res := make([]*v1alpha2.GitHubTeam, 0, len(teams))
	for _, team := range teams {
		teamUsers, err := t.descendants(ctx, team)
		if err != nil {
			retErr = errors.Join(retErr, err)
			// Skip as there is error when fetching descendants.
			continue
		}
		res = append(res, &v1alpha2.GitHubTeam{
			TeamId:        team.GetTeamId(),
			OrgId:         team.GetOrgId(),
			SourceTeamIds: team.GetSourceTeamIds(),
			Users:         teamUsers,
		})
	}
	return res, retErr
}

// descendants fetches all the descendants of the given team and maps them to GitHub users.
// Any duplicate GitHub users will be removed.
func (t *TeamLinkService) descendants(ctx context.Context, team *v1alpha2.GitHubTeam) ([]*v1alpha2.GitHubUser, error) {
	var teamUsers []*v1alpha2.GitHubUser
	var merr error
	for _, teamID := range team.GetSourceTeamIds() {
		users, err := t.sourceTeamClient.Descendants(ctx, teamID)
		if err != nil {
			merr = errors.Join(merr, fmt.Errorf("failed to get descendants for team ID %s: %w", teamID, err))
			// continue processing other teamIDs
			continue
		}
		githubUsers, err := t.githubUsers(ctx, users)
		if err != nil {
			merr = errors.Join(merr, fmt.Errorf("error getting some or all github users for source team ID %s: %w", teamID, err))
		}
		teamUsers = append(teamUsers, githubUsers...)
	}
	return uniqueUsers(teamUsers), merr
}

// githubUsers maps each source team user to its corresponding GitHub user and returns nil if an error occurs.
func (t *TeamLinkService) githubUsers(ctx context.Context, users []string) ([]*v1alpha2.GitHubUser, error) {
	githubUsers := make([]*v1alpha2.GitHubUser, 0, len(users))
	var merr error
	for _, user := range users {
		githubUser, err := t.githubMapper.GitHubUser(ctx, user)
		if err != nil {
			merr = errors.Join(merr, fmt.Errorf("error mapping source user to their github user %s: %w", user, err))
			// continue processing the other users
			continue
		}
		// Ignore if the user is not found.
		if githubUser != nil {
			githubUsers = append(githubUsers, githubUser)
		}
	}
	return githubUsers, merr
}

// uniqueUsers returns a list of unique GitHub users from the given list of users.
func uniqueUsers(users []*v1alpha2.GitHubUser) []*v1alpha2.GitHubUser {
	var uniqueUsers []*v1alpha2.GitHubUser
	userSet := make(map[string]struct{})
	for _, u := range users {
		if _, ok := userSet[u.GetEmail()]; !ok {
			userSet[u.GetEmail()] = struct{}{}
			uniqueUsers = append(uniqueUsers, u)
		}
	}
	return uniqueUsers
}
