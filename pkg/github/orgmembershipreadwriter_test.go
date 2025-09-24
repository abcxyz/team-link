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
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v67/github"
	"google.golang.org/protobuf/proto"

	"github.com/abcxyz/pkg/testutil"
	"github.com/abcxyz/team-link/v2/pkg/groupsync"
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
			wantErr: "could not parse orgID invalidID",
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
		opts        []OrgRWOpt
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
				orgMembers: map[string]map[string]*github.Membership{
					"8583": { // org1
						"user1": &github.Membership{Role: proto.String("member")},
						"user3": &github.Membership{Role: proto.String("member")},
					},
					"4701": { // org2
						"user2": &github.Membership{Role: proto.String("member")},
					},
				},
				invitations: map[string][]*github.Invitation{
					"8583": {
						&github.Invitation{
							ID:    proto.Int64(123),
							Login: proto.String("user2"),
							Email: proto.String("user2@example.com"),
							Role:  proto.String("admin"),
						},
					},
					"4701": {},
				},
			},
			groupID: "8583",
			want: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:       proto.Int64(2286),
							Login:    proto.String("user1"),
							Email:    proto.String("user1@example.com"),
							RoleName: proto.String("member"),
						},
						Metadata: &RoleMetadata{Role: Member},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user2",
						Attributes: &github.Invitation{
							ID:    proto.Int64(123),
							Login: proto.String("user2"),
							Email: proto.String("user2@example.com"),
							Role:  proto.String("admin"),
						},
						Metadata: &RoleMetadata{Role: Admin},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &github.User{
							ID:       proto.Int64(3208),
							Login:    proto.String("user3"),
							Email:    proto.String("user3@example.com"),
							RoleName: proto.String("member"),
						},
						Metadata: &RoleMetadata{Role: Member},
					},
				},
			},
		},
		{
			name: "success_ignore_invitations",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			opts: []OrgRWOpt{WithInvitations(false)},
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
				orgMembers: map[string]map[string]*github.Membership{
					"8583": { // org1
						"user1": &github.Membership{Role: proto.String("member")},
						"user3": &github.Membership{Role: proto.String("member")},
					},
					"4701": { // org2
						"user2": &github.Membership{Role: proto.String("member")},
					},
				},
				invitations: map[string][]*github.Invitation{
					"8583": {
						&github.Invitation{
							ID:    proto.Int64(123),
							Login: proto.String("user2"),
							Email: proto.String("user2@example.com"),
							Role:  proto.String("admin"),
						},
					},
					"4701": {},
				},
			},
			groupID: "8583",
			want: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:       proto.Int64(2286),
							Login:    proto.String("user1"),
							Email:    proto.String("user1@example.com"),
							RoleName: proto.String("member"),
						},
						Metadata: &RoleMetadata{Role: Member},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &github.User{
							ID:       proto.Int64(3208),
							Login:    proto.String("user3"),
							Email:    proto.String("user3@example.com"),
							RoleName: proto.String("member"),
						},
						Metadata: &RoleMetadata{Role: Member},
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
			wantErr: "could not parse orgID invalidID",
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
				orgMembers: map[string]map[string]*github.Membership{
					"8583": { // org1
						"user1": &github.Membership{Role: proto.String("member")},
						"user3": &github.Membership{Role: proto.String("member")},
					},
					"4701": { // org2
						"user2": &github.Membership{Role: proto.String("member")},
					},
				},
				invitations: map[string][]*github.Invitation{
					"8583": {},
					"4701": {},
				},
			},
			groupID: "8583",
			want: []*groupsync.User{
				{
					ID: "user1",
					Attributes: &github.User{
						ID:       proto.Int64(2286),
						Login:    proto.String("user1"),
						Email:    proto.String("user1@example.com"),
						RoleName: proto.String("member"),
					},
					Metadata: &RoleMetadata{Role: Member},
				},
				{
					ID: "user3",
					Attributes: &github.User{
						ID:       proto.Int64(3208),
						Login:    proto.String("user3"),
						Email:    proto.String("user3@example.com"),
						RoleName: proto.String("member"),
					},
					Metadata: &RoleMetadata{Role: Member},
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
			wantErr: "could not parse orgID invalidID",
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
		{
			name: "not_found",
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
				},
			},
			userID:  "fakeuser",
			wantErr: "could not get user",
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
		opts         []OrgRWOpt
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
				orgMembers: map[string]map[string]*github.Membership{
					"8583": { // org1
						"user1": &github.Membership{Role: proto.String("member")},
						"user3": &github.Membership{Role: proto.String("member")},
					},
					"4701": { // org2
						"user2": &github.Membership{Role: proto.String("member")},
					},
				},
				invitations: map[string][]*github.Invitation{
					"8583": {},
					"4701": {},
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
						Metadata: &RoleMetadata{Role: Member},
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
						Metadata: &RoleMetadata{Role: Admin},
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
						Metadata: &RoleMetadata{Role: Member},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:       proto.Int64(2286),
							Login:    proto.String("user1"),
							Email:    proto.String("user1@example.com"),
							RoleName: proto.String("member"),
						},
						Metadata: &RoleMetadata{Role: Member},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user2",
						Attributes: &github.Invitation{
							ID:    proto.Int64(1),
							Login: proto.String("user2"),
							Email: proto.String("user2@example.com"),
							Role:  proto.String("admin"),
						},
						Metadata: &RoleMetadata{Role: Admin},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &github.User{
							ID:       proto.Int64(3208),
							Login:    proto.String("user3"),
							Email:    proto.String("user3@example.com"),
							RoleName: proto.String("member"),
						},
						Metadata: &RoleMetadata{Role: Member},
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
				orgMembers: map[string]map[string]*github.Membership{
					"8583": { // org1
						"user2": &github.Membership{Role: proto.String("member")},
						"user1": &github.Membership{Role: proto.String("member")},
						"user3": &github.Membership{Role: proto.String("member")},
					},
					"4701": { // org2
						"user1": &github.Membership{Role: proto.String("member")},
					},
				},
				invitations: map[string][]*github.Invitation{
					"8583": {},
					"4701": {},
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
						Metadata: &RoleMetadata{Role: Member},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:       proto.Int64(2286),
							Login:    proto.String("user1"),
							Email:    proto.String("user1@example.com"),
							RoleName: proto.String("member"),
						},
						Metadata: &RoleMetadata{Role: Member},
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
				orgMembers: map[string]map[string]*github.Membership{
					"8583": { // org1
						"user1": &github.Membership{Role: proto.String("member")},
						"user3": &github.Membership{Role: proto.String("member")},
					},
					"4701": { // org2
						"user2": &github.Membership{Role: proto.String("member")},
					},
				},
				invitations: map[string][]*github.Invitation{
					"8583": {},
					"4701": {},
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
						Metadata: &RoleMetadata{Role: Member},
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
						Metadata: &RoleMetadata{Role: Member},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:       proto.Int64(2286),
							Login:    proto.String("user1"),
							Email:    proto.String("user1@example.com"),
							RoleName: proto.String("member"),
						},
						Metadata: &RoleMetadata{Role: Member},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user2",
						Attributes: &github.Invitation{
							ID:    proto.Int64(1),
							Login: proto.String("user2"),
							Email: proto.String("user2@example.com"),
							Role:  proto.String("direct_member"),
						},
						Metadata: &RoleMetadata{Role: Member},
					},
				},
			},
		},
		{
			name: "partial_failure_add_and_remove",
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
				orgMembers: map[string]map[string]*github.Membership{
					"8583": { // org1
						"user1": &github.Membership{Role: proto.String("member")},
						"user3": &github.Membership{Role: proto.String("member")},
					},
					"4701": { // org2
						"user2": &github.Membership{Role: proto.String("member")},
					},
				},
				invitations: map[string][]*github.Invitation{
					"8583": {},
					"4701": {},
				},
			},
			groupID: "8583",
			inputMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "fakeuser",
						Attributes: &github.User{
							ID:    proto.Int64(1234),
							Login: proto.String("fakeuser"),
							Email: proto.String("fakeuser@example.com"),
						},
						Metadata: &RoleMetadata{Role: Member},
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
						Metadata: &RoleMetadata{Role: Member},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user2",
						Attributes: &github.Invitation{
							ID:    proto.Int64(1),
							Login: proto.String("user2"),
							Email: proto.String("user2@example.com"),
							Role:  proto.String("direct_member"),
						},
						Metadata: &RoleMetadata{Role: Member},
					},
				},
			},
			wantSetErr: "failed to invite user(fakeuser)",
		},
		{
			name: "success_change_roles",
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
				orgMembers: map[string]map[string]*github.Membership{
					"8583": { // org1
						"user1": &github.Membership{Role: proto.String("admin")},
					},
					"4701": { // org2
						"user2": &github.Membership{Role: proto.String("member")},
					},
				},
				invitations: map[string][]*github.Invitation{
					"8583": {
						&github.Invitation{
							ID:    proto.Int64(1),
							Login: proto.String("user3"),
							Email: proto.String("user3@example.com"),
							Role:  proto.String("direct_member"),
						},
					},
					"4701": {},
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
						Metadata: &RoleMetadata{Role: Member},
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
						Metadata: &RoleMetadata{Role: Admin},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:       proto.Int64(2286),
							Login:    proto.String("user1"),
							Email:    proto.String("user1@example.com"),
							RoleName: proto.String("member"),
						},
						Metadata: &RoleMetadata{Role: Member},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &github.Invitation{
							ID:    proto.Int64(1),
							Login: proto.String("user3"),
							Email: proto.String("user3@example.com"),
							Role:  proto.String("admin"),
						},
						Metadata: &RoleMetadata{Role: Admin},
					},
				},
			},
		},
		{
			name: "success_without_invites",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			opts: []OrgRWOpt{WithInvitations(false)},
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
				orgMembers: map[string]map[string]*github.Membership{
					"8583": { // org1
						"user1": &github.Membership{Role: proto.String("admin")},
					},
					"4701": { // org2
						"user2": &github.Membership{Role: proto.String("member")},
					},
				},
				invitations: map[string][]*github.Invitation{
					"8583": {},
					"4701": {},
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
						Metadata: &RoleMetadata{Role: Member},
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
						Metadata: &RoleMetadata{Role: Admin},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:       proto.Int64(2286),
							Login:    proto.String("user1"),
							Email:    proto.String("user1@example.com"),
							RoleName: proto.String("member"),
						},
						Metadata: &RoleMetadata{Role: Member},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &github.User{
							ID:       proto.Int64(3208),
							Login:    proto.String("user3"),
							Email:    proto.String("user3@example.com"),
							RoleName: proto.String("admin"),
						},
						Metadata: &RoleMetadata{Role: Admin},
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
			wantGetErr: "could not parse orgID invalidID",
			wantSetErr: "could not parse orgID invalidID",
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
