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

package github

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v67/github"
	"google.golang.org/protobuf/proto"

	"github.com/abcxyz/pkg/testutil"
	"github.com/abcxyz/team-link/pkg/groupsync"
)

func TestOrgMembershipReadWriter_GetGroup(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		tokenSource OrgTokenSource
		data        *GitHubData
		groupID     string
		want        *groupsync.Group
		wantErr     string
	}{
		{
			name: "success",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data: &GitHubData{
				orgs: map[string]*github.Organization{
					"8583": {
						ID:   proto.Int64(8583),
						Name: proto.String("org1"),
					},
					"4701": {
						ID:   proto.Int64(4701),
						Name: proto.String("org2"),
					},
				},
			},
			groupID: "8583",
			want: &groupsync.Group{
				ID: "8583",
				Attributes: &github.Organization{
					ID:   proto.Int64(8583),
					Name: proto.String("org1"),
				},
			},
		},
		{
			name: "id_wrong_format",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data:    &GitHubData{},
			groupID: "invalidID",
			wantErr: "could not parse orgID from groupID: invalidID",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			server := fakeGitHub(tc.data)
			defer server.Close()

			client := githubClient(server)

			groupRW := NewOrgMembershipReadWriter(tc.tokenSource, client)

			got, err := groupRW.GetGroup(ctx, tc.groupID)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error : %v", err)
			}

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("unexpected gotMembers (-got, +want) = %v", diff)
			}
		})
	}
}

func TestOrgMembershipReadWriter_GetMembers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		tokenSource OrgTokenSource
		data        *GitHubData
		opts        []Opt
		groupID     string
		want        []groupsync.Member
		wantErr     string
	}{
		{
			name: "success",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data: &GitHubData{
				users: map[string]*github.User{
					"user1": {
						ID:    proto.Int64(2286),
						Login: proto.String("user1"),
						Email: proto.String("user1@example.com"),
					},
					"user2": {
						ID:    proto.Int64(5660),
						Login: proto.String("user2"),
						Email: proto.String("user2@example.com"),
					},
					"user3": {
						ID:    proto.Int64(3208),
						Login: proto.String("user3"),
						Email: proto.String("user3@example.com"),
					},
				},
				orgs: map[string]*github.Organization{
					"8583": {
						ID:   proto.Int64(8583),
						Name: proto.String("org1"),
					},
					"4701": {
						ID:   proto.Int64(4701),
						Name: proto.String("org2"),
					},
				},
				orgMembers: map[string]map[string]struct{}{
					"8583": { // org1
						"user1": struct{}{},
						"user3": struct{}{},
					},
					"4701": { // org2
						"user2": struct{}{},
					},
				},
			},
			groupID: "8583",
			want: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:    proto.Int64(2286),
							Login: proto.String("user1"),
							Email: proto.String("user1@example.com"),
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &github.User{
							ID:    proto.Int64(3208),
							Login: proto.String("user3"),
							Email: proto.String("user3@example.com"),
						},
					},
				},
			},
		},
		{
			name: "id_wrong_format",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data:    &GitHubData{},
			groupID: "invalidID",
			wantErr: "could not parse orgID from groupID: invalidID",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			server := fakeGitHub(tc.data)
			defer server.Close()

			client := githubClient(server)

			groupRW := NewOrgMembershipReadWriter(tc.tokenSource, client, tc.opts...)

			got, err := groupRW.GetMembers(ctx, tc.groupID)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error : %v", err)
			}

			// sort so we have a consistent ordering for comparison
			sortByID(got)

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("unexpected gotMembers (-got, +want) = %v", diff)
			}
		})
	}
}

func TestOrgMembershipReadWriter_GetDescendants(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		tokenSource OrgTokenSource
		data        *GitHubData
		groupID     string
		want        []*groupsync.User
		wantErr     string
	}{
		{
			name: "success",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data: &GitHubData{
				users: map[string]*github.User{
					"user1": {
						ID:    proto.Int64(2286),
						Login: proto.String("user1"),
						Email: proto.String("user1@example.com"),
					},
					"user2": {
						ID:    proto.Int64(5660),
						Login: proto.String("user2"),
						Email: proto.String("user2@example.com"),
					},
					"user3": {
						ID:    proto.Int64(3208),
						Login: proto.String("user3"),
						Email: proto.String("user3@example.com"),
					},
				},
				orgs: map[string]*github.Organization{
					"8583": {
						ID:   proto.Int64(8583),
						Name: proto.String("org1"),
					},
					"4701": {
						ID:   proto.Int64(4701),
						Name: proto.String("org2"),
					},
				},
				orgMembers: map[string]map[string]struct{}{
					"8583": { // org1
						"user1": struct{}{},
						"user3": struct{}{},
					},
					"4701": { // org2
						"user2": struct{}{},
					},
				},
			},
			groupID: "8583",
			want: []*groupsync.User{
				{
					ID: "user1",
					Attributes: &github.User{
						ID:    proto.Int64(2286),
						Login: proto.String("user1"),
						Email: proto.String("user1@example.com"),
					},
				},
				{
					ID: "user3",
					Attributes: &github.User{
						ID:    proto.Int64(3208),
						Login: proto.String("user3"),
						Email: proto.String("user3@example.com"),
					},
				},
			},
		},
		{
			name: "id_wrong_format",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data:    &GitHubData{},
			groupID: "invalidID",
			wantErr: "could not parse orgID from groupID: invalidID",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			server := fakeGitHub(tc.data)
			defer server.Close()

			client := githubClient(server)

			groupRW := NewOrgMembershipReadWriter(tc.tokenSource, client)

			got, err := groupRW.Descendants(ctx, tc.groupID)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error : %v", err)
			}

			// sort so we have a consistent ordering for comparison
			slices.SortFunc(got, func(a, b *groupsync.User) int {
				return strings.Compare(a.ID, b.ID)
			})

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("unexpected gotMembers (-got, +want) = %v", diff)
			}
		})
	}
}

func TestOrgMembershipReadWriter_GetUser(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		tokenSource OrgTokenSource
		data        *GitHubData
		userID      string
		want        *groupsync.User
		wantErr     string
	}{
		{
			name: "success",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data: &GitHubData{
				users: map[string]*github.User{
					"user1": {
						ID:    proto.Int64(2286),
						Login: proto.String("user1"),
						Email: proto.String("user1@example.com"),
					},
					"user2": {
						ID:    proto.Int64(5660),
						Login: proto.String("user2"),
						Email: proto.String("user2@example.com"),
					},
					"user3": {
						ID:    proto.Int64(3208),
						Login: proto.String("user3"),
						Email: proto.String("user3@example.com"),
					},
				},
			},
			userID: "user1",
			want: &groupsync.User{
				ID: "user1",
				Attributes: &github.User{
					ID:    proto.Int64(2286),
					Login: proto.String("user1"),
					Email: proto.String("user1@example.com"),
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			server := fakeGitHub(tc.data)
			defer server.Close()

			client := githubClient(server)

			groupRW := NewOrgMembershipReadWriter(tc.tokenSource, client)

			got, err := groupRW.GetUser(ctx, tc.userID)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error : %v", err)
			}

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("unexpected gotMembers (-got, +want) = %v", diff)
			}
		})
	}
}

func TestOrgMembershipReadWriter_SetMembers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		tokenSource  OrgTokenSource
		data         *GitHubData
		opts         []Opt
		groupID      string
		inputMembers []groupsync.Member
		wantMembers  []groupsync.Member
		wantSetErr   string
		wantGetErr   string
	}{
		{
			name: "success_add",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data: &GitHubData{
				users: map[string]*github.User{
					"user1": {
						ID:    proto.Int64(2286),
						Login: proto.String("user1"),
						Email: proto.String("user1@example.com"),
					},
					"user2": {
						ID:    proto.Int64(5660),
						Login: proto.String("user2"),
						Email: proto.String("user2@example.com"),
					},
					"user3": {
						ID:    proto.Int64(3208),
						Login: proto.String("user3"),
						Email: proto.String("user3@example.com"),
					},
				},
				orgs: map[string]*github.Organization{
					"8583": {
						ID:   proto.Int64(8583),
						Name: proto.String("org1"),
					},
					"4701": {
						ID:   proto.Int64(4701),
						Name: proto.String("org2"),
					},
				},
				orgMembers: map[string]map[string]struct{}{
					"8583": { // org1
						"user1": struct{}{},
						"user3": struct{}{},
					},
					"4701": { // org2
						"user2": struct{}{},
					},
				},
			},
			groupID: "8583",
			inputMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:    proto.Int64(2286),
							Login: proto.String("user1"),
							Email: proto.String("user1@example.com"),
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user2",
						Attributes: &github.User{
							ID:    proto.Int64(5660),
							Login: proto.String("user2"),
							Email: proto.String("user2@example.com"),
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &github.User{
							ID:    proto.Int64(3208),
							Login: proto.String("user3"),
							Email: proto.String("user3@example.com"),
						},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:    proto.Int64(2286),
							Login: proto.String("user1"),
							Email: proto.String("user1@example.com"),
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user2",
						Attributes: &github.User{
							ID:    proto.Int64(5660),
							Login: proto.String("user2"),
							Email: proto.String("user2@example.com"),
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &github.User{
							ID:    proto.Int64(3208),
							Login: proto.String("user3"),
							Email: proto.String("user3@example.com"),
						},
					},
				},
			},
		},
		{
			name: "success_remove",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data: &GitHubData{
				users: map[string]*github.User{
					"user1": {
						ID:    proto.Int64(2286),
						Login: proto.String("user1"),
						Email: proto.String("user1@example.com"),
					},
					"user2": {
						ID:    proto.Int64(5660),
						Login: proto.String("user2"),
						Email: proto.String("user2@example.com"),
					},
					"user3": {
						ID:    proto.Int64(3208),
						Login: proto.String("user3"),
						Email: proto.String("user3@example.com"),
					},
				},
				orgs: map[string]*github.Organization{
					"8583": {
						ID:   proto.Int64(8583),
						Name: proto.String("org1"),
					},
					"4701": {
						ID:   proto.Int64(4701),
						Name: proto.String("org2"),
					},
				},
				orgMembers: map[string]map[string]struct{}{
					"8583": { // org1
						"user2": struct{}{},
						"user1": struct{}{},
						"user3": struct{}{},
					},
					"4701": { // org2
						"user1": struct{}{},
					},
				},
			},
			groupID: "8583",
			inputMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:    proto.Int64(2286),
							Login: proto.String("user1"),
							Email: proto.String("user1@example.com"),
						},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:    proto.Int64(2286),
							Login: proto.String("user1"),
							Email: proto.String("user1@example.com"),
						},
					},
				},
			},
		},
		{
			name: "success_add_and_remove",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data: &GitHubData{
				users: map[string]*github.User{
					"user1": {
						ID:    proto.Int64(2286),
						Login: proto.String("user1"),
						Email: proto.String("user1@example.com"),
					},
					"user2": {
						ID:    proto.Int64(5660),
						Login: proto.String("user2"),
						Email: proto.String("user2@example.com"),
					},
					"user3": {
						ID:    proto.Int64(3208),
						Login: proto.String("user3"),
						Email: proto.String("user3@example.com"),
					},
				},
				orgs: map[string]*github.Organization{
					"8583": {
						ID:   proto.Int64(8583),
						Name: proto.String("org1"),
					},
					"4701": {
						ID:   proto.Int64(4701),
						Name: proto.String("org2"),
					},
				},
				orgMembers: map[string]map[string]struct{}{
					"8583": { // org1
						"user1": struct{}{},
						"user3": struct{}{},
					},
					"4701": { // org2
						"user2": struct{}{},
					},
				},
			},
			groupID: "8583",
			inputMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:    proto.Int64(2286),
							Login: proto.String("user1"),
							Email: proto.String("user1@example.com"),
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user2",
						Attributes: &github.User{
							ID:    proto.Int64(5660),
							Login: proto.String("user2"),
							Email: proto.String("user2@example.com"),
						},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:    proto.Int64(2286),
							Login: proto.String("user1"),
							Email: proto.String("user1@example.com"),
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user2",
						Attributes: &github.User{
							ID:    proto.Int64(5660),
							Login: proto.String("user2"),
							Email: proto.String("user2@example.com"),
						},
					},
				},
			},
		},
		{
			name: "id_wrong_format",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data:       &GitHubData{},
			groupID:    "invalidID",
			wantGetErr: "could not parse orgID from groupID: invalidID",
			wantSetErr: "could not parse orgID from groupID: invalidID",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			server := fakeGitHub(tc.data)
			defer server.Close()

			client := githubClient(server)

			groupRW := NewOrgMembershipReadWriter(tc.tokenSource, client, tc.opts...)

			err := groupRW.SetMembers(ctx, tc.groupID, tc.inputMembers)
			if diff := testutil.DiffErrString(err, tc.wantSetErr); diff != "" {
				t.Errorf("unexpected error (-got, +want) = %v", diff)
			}

			gotMembers, err := groupRW.GetMembers(ctx, tc.groupID)
			if diff := testutil.DiffErrString(err, tc.wantGetErr); diff != "" {
				t.Errorf("unexpected error : %v", err)
			}

			// sort so we have a consistent ordering for comparison
			sortByID(gotMembers)

			if diff := cmp.Diff(gotMembers, tc.wantMembers); diff != "" {
				t.Errorf("unexpected gotMembers (-got, +want) = %v", diff)
			}
		})
	}
}
