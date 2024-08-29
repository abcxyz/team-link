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

package teamlink

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/abcxyz/pkg/testutil"
	api "github.com/abcxyz/team-link/apis/v1alpha2"
)

func TestTeamLinkService_SyncTeam(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name             string
		sourceTeamClient api.SourceTeamClient
		githubMapper     api.GitHubMapper
		syncerFuncErr    error
		syncer           *testSynchronizer
		teamID           string
		want             []*api.GitHubTeam
		wantErr          string
	}{
		{
			name: "successful_sync",
			sourceTeamClient: &testSourceTeamClient{
				descendants: map[string][]string{
					"sourceTeam1": {"sourceUser1", "sourceUser2"},
					"sourceTeam2": {"sourceUser3", "sourceUser4"},
					"sourceTeam3": {"sourceUser1", "sourceUser4"},
				},
			},
			githubMapper: &testGitHubMapper{
				userMap: map[string]*api.GitHubUser{
					"sourceUser1": {
						UserId: 56489,
						Login:  "githubUser1",
						Email:  "sourceUser1@example.com",
					},
					"sourceUser2": {
						UserId: 22009,
						Login:  "githubUser2",
						Email:  "sourceUser2@example.com",
					},
					"sourceUser3": {
						UserId: 36397,
						Login:  "githubUser3",
						Email:  "sourceUser3@example.com",
					},
					"sourceUser4": {
						UserId: 75154,
						Login:  "githubUser4",
						Email:  "sourceUser4@example.com",
					},
				},
				teamMap: map[string][]*api.GitHubTeam{
					"sourceTeam1": {
						{
							TeamId:        42455,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam1"},
						},
					},
					"sourceTeam2": {
						{
							TeamId:        64134,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam2"},
						},
					},
					"sourceTeam3": {
						{
							TeamId:        65286,
							OrgId:         77172,
							SourceTeamIds: []string{"sourceTeam3"},
						},
					},
				},
			},
			syncer: &testSynchronizer{},
			teamID: "sourceTeam1",
			want: []*api.GitHubTeam{
				{
					TeamId:        42455,
					OrgId:         80703,
					SourceTeamIds: []string{"sourceTeam1"},
					Users: []*api.GitHubUser{
						{
							UserId: 56489,
							Login:  "githubUser1",
							Email:  "sourceUser1@example.com",
						},
						{
							UserId: 22009,
							Login:  "githubUser2",
							Email:  "sourceUser2@example.com",
						},
					},
				},
			},
		},
		{
			name: "syncerFunc_error",
			sourceTeamClient: &testSourceTeamClient{
				descendants: map[string][]string{
					"sourceTeam1": {"sourceUser1", "sourceUser2"},
					"sourceTeam2": {"sourceUser3", "sourceUser4"},
					"sourceTeam3": {"sourceUser1", "sourceUser4"},
				},
			},
			githubMapper: &testGitHubMapper{
				userMap: map[string]*api.GitHubUser{
					"sourceUser1": {
						UserId: 56489,
						Login:  "githubUser1",
						Email:  "sourceUser1@example.com",
					},
					"sourceUser2": {
						UserId: 22009,
						Login:  "githubUser2",
						Email:  "sourceUser2@example.com",
					},
					"sourceUser3": {
						UserId: 36397,
						Login:  "githubUser3",
						Email:  "sourceUser3@example.com",
					},
					"sourceUser4": {
						UserId: 75154,
						Login:  "githubUser4",
						Email:  "sourceUser4@example.com",
					},
				},
				teamMap: map[string][]*api.GitHubTeam{
					"sourceTeam1": {
						{
							TeamId:        42455,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam1"},
						},
					},
					"sourceTeam2": {
						{
							TeamId:        64134,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam2"},
						},
					},
					"sourceTeam3": {
						{
							TeamId:        65286,
							OrgId:         77172,
							SourceTeamIds: []string{"sourceTeam3"},
						},
					},
				},
			},
			syncerFuncErr: fmt.Errorf("syncerFunc error"),
			syncer:        &testSynchronizer{},
			teamID:        "sourceTeam1",
			wantErr:       "failed to get syncer: syncerFunc error",
		},
		{
			name: "syncer_error",
			sourceTeamClient: &testSourceTeamClient{
				descendants: map[string][]string{
					"sourceTeam1": {"sourceUser1", "sourceUser2"},
					"sourceTeam2": {"sourceUser3", "sourceUser4"},
					"sourceTeam3": {"sourceUser1", "sourceUser4"},
				},
			},
			githubMapper: &testGitHubMapper{
				userMap: map[string]*api.GitHubUser{
					"sourceUser1": {
						UserId: 56489,
						Login:  "githubUser1",
						Email:  "sourceUser1@example.com",
					},
					"sourceUser2": {
						UserId: 22009,
						Login:  "githubUser2",
						Email:  "sourceUser2@example.com",
					},
					"sourceUser3": {
						UserId: 36397,
						Login:  "githubUser3",
						Email:  "sourceUser3@example.com",
					},
					"sourceUser4": {
						UserId: 75154,
						Login:  "githubUser4",
						Email:  "sourceUser4@example.com",
					},
				},
				teamMap: map[string][]*api.GitHubTeam{
					"sourceTeam1": {
						{
							TeamId:        42455,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam1"},
						},
					},
					"sourceTeam2": {
						{
							TeamId:        64134,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam2"},
						},
					},
					"sourceTeam3": {
						{
							TeamId:        65286,
							OrgId:         77172,
							SourceTeamIds: []string{"sourceTeam3"},
						},
					},
				},
			},
			syncer: &testSynchronizer{
				err: fmt.Errorf("syncer error"),
			},
			teamID:  "sourceTeam1",
			wantErr: "failed to sync some/all teams: syncer error",
		},
		{
			name: "sourceTeamClient_error",
			sourceTeamClient: &testSourceTeamClient{
				err: fmt.Errorf("sourceTeamClient error"),
			},
			githubMapper: &testGitHubMapper{
				userMap: map[string]*api.GitHubUser{
					"sourceUser1": {
						UserId: 56489,
						Login:  "githubUser1",
						Email:  "sourceUser1@example.com",
					},
					"sourceUser2": {
						UserId: 22009,
						Login:  "githubUser2",
						Email:  "sourceUser2@example.com",
					},
					"sourceUser3": {
						UserId: 36397,
						Login:  "githubUser3",
						Email:  "sourceUser3@example.com",
					},
					"sourceUser4": {
						UserId: 75154,
						Login:  "githubUser4",
						Email:  "sourceUser4@example.com",
					},
				},
				teamMap: map[string][]*api.GitHubTeam{
					"sourceTeam1": {
						{
							TeamId:        42455,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam1"},
						},
					},
					"sourceTeam2": {
						{
							TeamId:        64134,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam2"},
						},
					},
					"sourceTeam3": {
						{
							TeamId:        65286,
							OrgId:         77172,
							SourceTeamIds: []string{"sourceTeam3"},
						},
					},
				},
			},
			syncer:  &testSynchronizer{},
			teamID:  "sourceTeam1",
			wantErr: "sourceTeamClient error",
		},
		{
			name: "githubMapper_map_user_error",
			sourceTeamClient: &testSourceTeamClient{
				descendants: map[string][]string{
					"sourceTeam1": {"sourceUser1", "sourceUser2"},
					"sourceTeam2": {"sourceUser3", "sourceUser4"},
					"sourceTeam3": {"sourceUser1", "sourceUser4"},
				},
			},
			githubMapper: &testGitHubMapper{
				userMap: map[string]*api.GitHubUser{
					"sourceUser1": {
						UserId: 56489,
						Login:  "githubUser1",
						Email:  "sourceUser1@example.com",
					},
					"sourceUser2": {
						UserId: 22009,
						Login:  "githubUser2",
						Email:  "sourceUser2@example.com",
					},
					"sourceUser3": {
						UserId: 36397,
						Login:  "githubUser3",
						Email:  "sourceUser3@example.com",
					},
					"sourceUser4": {
						UserId: 75154,
						Login:  "githubUser4",
						Email:  "sourceUser4@example.com",
					},
				},
				teamMap: map[string][]*api.GitHubTeam{
					"sourceTeam1": {
						{
							TeamId:        42455,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam1"},
						},
					},
					"sourceTeam2": {
						{
							TeamId:        64134,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam2"},
						},
					},
					"sourceTeam3": {
						{
							TeamId:        65286,
							OrgId:         77172,
							SourceTeamIds: []string{"sourceTeam3"},
						},
					},
				},
				githubUserErrs: map[string]error{
					"sourceUser1": fmt.Errorf("error mapping sourceUser1"),
				},
			},
			syncer:  &testSynchronizer{},
			teamID:  "sourceTeam1",
			wantErr: "error getting some or all github users for source team ID sourceTeam1: error mapping source user to their github user sourceUser1: error mapping sourceUser1",
		},
		{
			name: "githubMapper_map_team_error",
			sourceTeamClient: &testSourceTeamClient{
				descendants: map[string][]string{
					"sourceTeam1": {"sourceUser1", "sourceUser2"},
					"sourceTeam2": {"sourceUser3", "sourceUser4"},
					"sourceTeam3": {"sourceUser1", "sourceUser4"},
				},
			},
			githubMapper: &testGitHubMapper{
				userMap: map[string]*api.GitHubUser{
					"sourceUser1": {
						UserId: 56489,
						Login:  "githubUser1",
						Email:  "sourceUser1@example.com",
					},
					"sourceUser2": {
						UserId: 22009,
						Login:  "githubUser2",
						Email:  "sourceUser2@example.com",
					},
					"sourceUser3": {
						UserId: 36397,
						Login:  "githubUser3",
						Email:  "sourceUser3@example.com",
					},
					"sourceUser4": {
						UserId: 75154,
						Login:  "githubUser4",
						Email:  "sourceUser4@example.com",
					},
				},
				teamMap: map[string][]*api.GitHubTeam{
					"sourceTeam1": {
						{
							TeamId:        42455,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam1"},
						},
					},
					"sourceTeam2": {
						{
							TeamId:        64134,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam2"},
						},
					},
					"sourceTeam3": {
						{
							TeamId:        65286,
							OrgId:         77172,
							SourceTeamIds: []string{"sourceTeam3"},
						},
					},
				},
				githubTeamsErrs: map[string]error{
					"sourceTeam1": fmt.Errorf("error mapping sourceTeam1"),
				},
			},
			syncer:  &testSynchronizer{},
			teamID:  "sourceTeam1",
			wantErr: "error mapping sourceTeam1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			syncerFunc := func(ctx context.Context) (api.TeamSynchronizer, error) {
				if tc.syncerFuncErr != nil {
					return nil, tc.syncerFuncErr
				}
				return tc.syncer, nil
			}
			teamLinkService := New(tc.sourceTeamClient, tc.githubMapper, syncerFunc)
			err := teamLinkService.SyncTeam(ctx, &api.SourceEvent{TeamId: tc.teamID})
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error (-got, +want):\n%s", diff)
			}
			// sort tc.syncer.requestSyncs so we have a consistent ordering for comparison
			sort.Slice(tc.syncer.requestSyncs, func(i, j int) bool {
				return tc.syncer.requestSyncs[i].GetTeamId() < tc.syncer.requestSyncs[j].GetTeamId()
			})
			if diff := cmp.Diff(tc.syncer.requestSyncs, tc.want, protocmp.Transform()); diff != "" {
				t.Errorf("unexpected request (-got, +want):\n%s", diff)
			}
		})
	}
}

func TestTeamLinkService_SyncAll(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name             string
		sourceTeamClient api.SourceTeamClient
		githubMapper     api.GitHubMapper
		syncerFuncErr    error
		syncer           *testSynchronizer
		want             []*api.GitHubTeam
		wantErr          string
	}{
		{
			name: "successful_sync",
			sourceTeamClient: &testSourceTeamClient{
				descendants: map[string][]string{
					"sourceTeam1": {"sourceUser1", "sourceUser2"},
					"sourceTeam2": {"sourceUser3", "sourceUser4"},
					"sourceTeam3": {"sourceUser1", "sourceUser4"},
				},
			},
			githubMapper: &testGitHubMapper{
				userMap: map[string]*api.GitHubUser{
					"sourceUser1": {
						UserId: 56489,
						Login:  "githubUser1",
						Email:  "sourceUser1@example.com",
					},
					"sourceUser2": {
						UserId: 22009,
						Login:  "githubUser2",
						Email:  "sourceUser2@example.com",
					},
					"sourceUser3": {
						UserId: 36397,
						Login:  "githubUser3",
						Email:  "sourceUser3@example.com",
					},
					"sourceUser4": {
						UserId: 75154,
						Login:  "githubUser4",
						Email:  "sourceUser4@example.com",
					},
				},
				teamMap: map[string][]*api.GitHubTeam{
					"sourceTeam1": {
						{
							TeamId:        42455,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam1"},
						},
					},
					"sourceTeam2": {
						{
							TeamId:        64134,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam2"},
						},
					},
					"sourceTeam3": {
						{
							TeamId:        65286,
							OrgId:         77172,
							SourceTeamIds: []string{"sourceTeam3"},
						},
					},
				},
			},
			syncer: &testSynchronizer{},
			want: []*api.GitHubTeam{
				{
					TeamId:        42455,
					OrgId:         80703,
					SourceTeamIds: []string{"sourceTeam1"},
					Users: []*api.GitHubUser{
						{
							UserId: 56489,
							Login:  "githubUser1",
							Email:  "sourceUser1@example.com",
						},
						{
							UserId: 22009,
							Login:  "githubUser2",
							Email:  "sourceUser2@example.com",
						},
					},
				},
				{
					TeamId:        64134,
					OrgId:         80703,
					SourceTeamIds: []string{"sourceTeam2"},
					Users: []*api.GitHubUser{
						{
							UserId: 36397,
							Login:  "githubUser3",
							Email:  "sourceUser3@example.com",
						},
						{
							UserId: 75154,
							Login:  "githubUser4",
							Email:  "sourceUser4@example.com",
						},
					},
				},
				{
					TeamId:        65286,
					OrgId:         77172,
					SourceTeamIds: []string{"sourceTeam3"},
					Users: []*api.GitHubUser{
						{
							UserId: 56489,
							Login:  "githubUser1",
							Email:  "sourceUser1@example.com",
						},
						{
							UserId: 75154,
							Login:  "githubUser4",
							Email:  "sourceUser4@example.com",
						},
					},
				},
			},
		},
		{
			name: "syncerFunc_error",
			sourceTeamClient: &testSourceTeamClient{
				descendants: map[string][]string{
					"sourceTeam1": {"sourceUser1", "sourceUser2"},
					"sourceTeam2": {"sourceUser3", "sourceUser4"},
					"sourceTeam3": {"sourceUser1", "sourceUser4"},
				},
			},
			githubMapper: &testGitHubMapper{
				userMap: map[string]*api.GitHubUser{
					"sourceUser1": {
						UserId: 56489,
						Login:  "githubUser1",
						Email:  "sourceUser1@example.com",
					},
					"sourceUser2": {
						UserId: 22009,
						Login:  "githubUser2",
						Email:  "sourceUser2@example.com",
					},
					"sourceUser3": {
						UserId: 36397,
						Login:  "githubUser3",
						Email:  "sourceUser3@example.com",
					},
					"sourceUser4": {
						UserId: 75154,
						Login:  "githubUser4",
						Email:  "sourceUser4@example.com",
					},
				},
				teamMap: map[string][]*api.GitHubTeam{
					"sourceTeam1": {
						{
							TeamId:        42455,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam1"},
						},
					},
					"sourceTeam2": {
						{
							TeamId:        64134,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam2"},
						},
					},
					"sourceTeam3": {
						{
							TeamId:        65286,
							OrgId:         77172,
							SourceTeamIds: []string{"sourceTeam3"},
						},
					},
				},
			},
			syncerFuncErr: fmt.Errorf("syncerFunc error"),
			syncer:        &testSynchronizer{},
			wantErr:       "failed to get syncer: syncerFunc error",
		},
		{
			name: "syncer_error",
			sourceTeamClient: &testSourceTeamClient{
				descendants: map[string][]string{
					"sourceTeam1": {"sourceUser1", "sourceUser2"},
					"sourceTeam2": {"sourceUser3", "sourceUser4"},
					"sourceTeam3": {"sourceUser1", "sourceUser4"},
				},
			},
			githubMapper: &testGitHubMapper{
				userMap: map[string]*api.GitHubUser{
					"sourceUser1": {
						UserId: 56489,
						Login:  "githubUser1",
						Email:  "sourceUser1@example.com",
					},
					"sourceUser2": {
						UserId: 22009,
						Login:  "githubUser2",
						Email:  "sourceUser2@example.com",
					},
					"sourceUser3": {
						UserId: 36397,
						Login:  "githubUser3",
						Email:  "sourceUser3@example.com",
					},
					"sourceUser4": {
						UserId: 75154,
						Login:  "githubUser4",
						Email:  "sourceUser4@example.com",
					},
				},
				teamMap: map[string][]*api.GitHubTeam{
					"sourceTeam1": {
						{
							TeamId:        42455,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam1"},
						},
					},
					"sourceTeam2": {
						{
							TeamId:        64134,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam2"},
						},
					},
					"sourceTeam3": {
						{
							TeamId:        65286,
							OrgId:         77172,
							SourceTeamIds: []string{"sourceTeam3"},
						},
					},
				},
			},
			syncer: &testSynchronizer{
				err: fmt.Errorf("syncer error"),
			},
			wantErr: "failed to sync some/all teams: syncer error",
		},
		{
			name: "sourceTeamClient_error",
			sourceTeamClient: &testSourceTeamClient{
				err: fmt.Errorf("sourceTeamClient error"),
			},
			githubMapper: &testGitHubMapper{
				userMap: map[string]*api.GitHubUser{
					"sourceUser1": {
						UserId: 56489,
						Login:  "githubUser1",
						Email:  "sourceUser1@example.com",
					},
					"sourceUser2": {
						UserId: 22009,
						Login:  "githubUser2",
						Email:  "sourceUser2@example.com",
					},
					"sourceUser3": {
						UserId: 36397,
						Login:  "githubUser3",
						Email:  "sourceUser3@example.com",
					},
					"sourceUser4": {
						UserId: 75154,
						Login:  "githubUser4",
						Email:  "sourceUser4@example.com",
					},
				},
				teamMap: map[string][]*api.GitHubTeam{
					"sourceTeam1": {
						{
							TeamId:        42455,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam1"},
						},
					},
					"sourceTeam2": {
						{
							TeamId:        64134,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam2"},
						},
					},
					"sourceTeam3": {
						{
							TeamId:        65286,
							OrgId:         77172,
							SourceTeamIds: []string{"sourceTeam3"},
						},
					},
				},
			},
			syncer:  &testSynchronizer{},
			wantErr: "sourceTeamClient error",
		},
		{
			name: "githubMapper_map_user_error",
			sourceTeamClient: &testSourceTeamClient{
				descendants: map[string][]string{
					"sourceTeam1": {"sourceUser1", "sourceUser2"},
					"sourceTeam2": {"sourceUser3", "sourceUser4"},
					"sourceTeam3": {"sourceUser1", "sourceUser4"},
				},
			},
			githubMapper: &testGitHubMapper{
				userMap: map[string]*api.GitHubUser{
					"sourceUser1": {
						UserId: 56489,
						Login:  "githubUser1",
						Email:  "sourceUser1@example.com",
					},
					"sourceUser2": {
						UserId: 22009,
						Login:  "githubUser2",
						Email:  "sourceUser2@example.com",
					},
					"sourceUser3": {
						UserId: 36397,
						Login:  "githubUser3",
						Email:  "sourceUser3@example.com",
					},
					"sourceUser4": {
						UserId: 75154,
						Login:  "githubUser4",
						Email:  "sourceUser4@example.com",
					},
				},
				teamMap: map[string][]*api.GitHubTeam{
					"sourceTeam1": {
						{
							TeamId:        42455,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam1"},
						},
					},
					"sourceTeam2": {
						{
							TeamId:        64134,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam2"},
						},
					},
					"sourceTeam3": {
						{
							TeamId:        65286,
							OrgId:         77172,
							SourceTeamIds: []string{"sourceTeam3"},
						},
					},
				},
				githubUserErrs: map[string]error{
					"sourceUser1": fmt.Errorf("error mapping sourceUser1"),
				},
			},
			syncer:  &testSynchronizer{},
			wantErr: "error getting some or all github users for source team ID sourceTeam1: error mapping source user to their github user sourceUser1: error mapping sourceUser1",
			want: []*api.GitHubTeam{
				{
					TeamId:        64134,
					OrgId:         80703,
					SourceTeamIds: []string{"sourceTeam2"},
					Users: []*api.GitHubUser{
						{
							UserId: 36397,
							Login:  "githubUser3",
							Email:  "sourceUser3@example.com",
						},
						{
							UserId: 75154,
							Login:  "githubUser4",
							Email:  "sourceUser4@example.com",
						},
					},
				},
			},
		},
		{
			name: "githubMapper_map_team_error",
			sourceTeamClient: &testSourceTeamClient{
				descendants: map[string][]string{
					"sourceTeam1": {"sourceUser1", "sourceUser2"},
					"sourceTeam2": {"sourceUser3", "sourceUser4"},
					"sourceTeam3": {"sourceUser1", "sourceUser4"},
				},
			},
			githubMapper: &testGitHubMapper{
				userMap: map[string]*api.GitHubUser{
					"sourceUser1": {
						UserId: 56489,
						Login:  "githubUser1",
						Email:  "sourceUser1@example.com",
					},
					"sourceUser2": {
						UserId: 22009,
						Login:  "githubUser2",
						Email:  "sourceUser2@example.com",
					},
					"sourceUser3": {
						UserId: 36397,
						Login:  "githubUser3",
						Email:  "sourceUser3@example.com",
					},
					"sourceUser4": {
						UserId: 75154,
						Login:  "githubUser4",
						Email:  "sourceUser4@example.com",
					},
				},
				teamMap: map[string][]*api.GitHubTeam{
					"sourceTeam1": {
						{
							TeamId:        42455,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam1"},
						},
					},
					"sourceTeam2": {
						{
							TeamId:        64134,
							OrgId:         80703,
							SourceTeamIds: []string{"sourceTeam2"},
						},
					},
					"sourceTeam3": {
						{
							TeamId:        65286,
							OrgId:         77172,
							SourceTeamIds: []string{"sourceTeam3"},
						},
					},
				},
				githubTeamsErrs: map[string]error{
					"sourceTeam1": fmt.Errorf("error mapping sourceTeam1"),
				},
			},
			syncer:  &testSynchronizer{},
			wantErr: "error mapping sourceTeam1",
			want: []*api.GitHubTeam{
				{
					TeamId:        64134,
					OrgId:         80703,
					SourceTeamIds: []string{"sourceTeam2"},
					Users: []*api.GitHubUser{
						{
							UserId: 36397,
							Login:  "githubUser3",
							Email:  "sourceUser3@example.com",
						},
						{
							UserId: 75154,
							Login:  "githubUser4",
							Email:  "sourceUser4@example.com",
						},
					},
				},
				{
					TeamId:        65286,
					OrgId:         77172,
					SourceTeamIds: []string{"sourceTeam3"},
					Users: []*api.GitHubUser{
						{
							UserId: 56489,
							Login:  "githubUser1",
							Email:  "sourceUser1@example.com",
						},
						{
							UserId: 75154,
							Login:  "githubUser4",
							Email:  "sourceUser4@example.com",
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			syncerFunc := func(ctx context.Context) (api.TeamSynchronizer, error) {
				if tc.syncerFuncErr != nil {
					return nil, tc.syncerFuncErr
				}
				return tc.syncer, nil
			}
			teamLinkService := New(tc.sourceTeamClient, tc.githubMapper, syncerFunc)
			err := teamLinkService.SyncAllTeams(ctx)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error (-got, +want):\n%s", diff)
			}
			// sort tc.syncer.requestSyncs so we have a consistent ordering for comparison
			sort.Slice(tc.syncer.requestSyncs, func(i, j int) bool {
				return tc.syncer.requestSyncs[i].GetTeamId() < tc.syncer.requestSyncs[j].GetTeamId()
			})
			if diff := cmp.Diff(tc.syncer.requestSyncs, tc.want, protocmp.Transform()); diff != "" {
				t.Errorf("unexpected request (-got, +want):\n%s", diff)
				t.Log("requestedSyncs:")
				for _, team := range tc.syncer.requestSyncs {
					t.Logf("team: %s", team)
				}
			}
		})
	}
}

type testSourceTeamClient struct {
	descendants map[string][]string
	err         error
}

func (tsc *testSourceTeamClient) Descendants(ctx context.Context, sourceTeamID string) ([]string, error) {
	if tsc.err != nil {
		return nil, tsc.err
	}
	return tsc.descendants[sourceTeamID], nil
}

type testGitHubMapper struct {
	userMap         map[string]*api.GitHubUser
	teamMap         map[string][]*api.GitHubTeam
	githubUserErrs  map[string]error
	githubTeamsErrs map[string]error
}

func (tgm *testGitHubMapper) GitHubUser(ctx context.Context, sourceUserID string) (*api.GitHubUser, error) {
	if err, ok := tgm.githubUserErrs[sourceUserID]; ok {
		return nil, err
	}
	user, ok := tgm.userMap[sourceUserID]
	if !ok {
		return nil, fmt.Errorf("no entry for user %s", sourceUserID)
	}
	return user, nil
}

func (tgm *testGitHubMapper) GitHubTeams(ctx context.Context, sourceTeamID string) ([]*api.GitHubTeam, error) {
	if err, ok := tgm.githubTeamsErrs[sourceTeamID]; ok {
		return nil, err
	}
	teams, ok := tgm.teamMap[sourceTeamID]
	if !ok {
		return nil, fmt.Errorf("no entry for team %s", sourceTeamID)
	}
	return teams, nil
}

func (tgm *testGitHubMapper) ContainsMappingForTeamID(ctx context.Context, sourceTeamID string) bool {
	_, ok := tgm.teamMap[sourceTeamID]
	return ok
}

// SourceTeamIDs returns the list of source team IDs for which this mapper has mappings.
func (tgm *testGitHubMapper) SourceTeamIDs(ctx context.Context) []string {
	sourceTeamIDs := make([]string, 0, len(tgm.teamMap))
	for teamID := range tgm.teamMap {
		sourceTeamIDs = append(sourceTeamIDs, teamID)
	}
	return sourceTeamIDs
}

type testSynchronizer struct {
	requestSyncs []*api.GitHubTeam
	err          error
}

func (ts *testSynchronizer) Sync(ctx context.Context, teams []*api.GitHubTeam) error {
	if ts.err != nil {
		return ts.err
	}
	ts.requestSyncs = append(ts.requestSyncs, teams...)
	return nil
}
