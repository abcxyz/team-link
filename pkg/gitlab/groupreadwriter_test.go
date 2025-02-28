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

package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/abcxyz/pkg/testutil"
	"github.com/abcxyz/team-link/pkg/groupsync"
)

func TestGroupReadWriter_GetGroup(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		data    *GitLabData
		groupID string
		want    *groupsync.Group
		wantErr string
	}{
		{
			name: "success",
			data: &GitLabData{
				groups: map[string]*gitlab.Group{
					"1": {
						ID:   1,
						Name: "group1",
					},
					"2": {
						ID:   2,
						Name: "group2",
					},
				},
			},
			groupID: "1",
			want: &groupsync.Group{
				ID: "1",
				Attributes: &gitlab.Group{
					ID:   1,
					Name: "group1",
				},
			},
		},
		{
			name:    "invalid_id",
			data:    &GitLabData{},
			groupID: "invalidID",
			wantErr: "failed to fetch group invalidID",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			server := fakeGitLab(tc.data)
			defer server.Close()

			clientProvider := gitlabClientProvider(server)
			groupRW := NewGroupReadWriter(clientProvider)

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

func TestGroupReadWriter_GetMembers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		data    *GitLabData
		opts    []Opt
		groupID string
		want    []groupsync.Member
		wantErr string
	}{
		{
			name: "success",
			data: &GitLabData{
				users: map[string]*gitlab.User{
					"user1": {
						ID:       2286,
						Username: "user1",
						Email:    "user1@example.com",
					},
					"user2": {
						ID:       5660,
						Username: "user2",
						Email:    "user2@example.com",
					},
					"user3": {
						ID:       3208,
						Username: "user3",
						Email:    "user3@example.com",
					},
				},
				groups: map[string]*gitlab.Group{
					"1": {
						ID:   1,
						Name: "group1",
					},
					"2": {
						ID:   2,
						Name: "group2",
					},
					"3": {
						ID:   3,
						Name: "group3",
					},
				},
				groupMembers: map[string]map[string]gitlab.AccessLevelValue{
					"1": {
						"user2": gitlab.DeveloperPermissions,
					},
					"2": {
						"user1": gitlab.DeveloperPermissions,
						"user3": gitlab.DeveloperPermissions,
					},
					"3": {
						"user1": gitlab.DeveloperPermissions,
					},
				},
				subgroups: map[string]map[string]struct{}{
					"1": {},
					"2": {},
					"3": {},
				},
			},
			groupID: "2",
			want: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.GroupMember{
							ID:          2286,
							Username:    "user1",
							Email:       "user1@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &gitlab.GroupMember{
							ID:          3208,
							Username:    "user3",
							Email:       "user3@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
			},
		},
		{
			name:    "invalid_id",
			data:    &GitLabData{},
			groupID: "invalidID",
			wantErr: "failed to fetch group members for invalidID",
		},
		{
			name: "subgroups_are_included",
			data: &GitLabData{
				users: map[string]*gitlab.User{
					"user1": {
						ID:       2286,
						Username: "user1",
						Email:    "user1@example.com",
					},
					"user2": {
						ID:       5660,
						Username: "user2",
						Email:    "user2@example.com",
					},
					"user3": {
						ID:       3208,
						Username: "user3",
						Email:    "user3@example.com",
					},
				},
				groups: map[string]*gitlab.Group{
					"1": {
						ID:   1,
						Name: "group1",
					},
					"2": {
						ID:   2,
						Name: "group2",
					},
					"4": {
						ID:       4,
						Name:     "sub",
						ParentID: 2,
					},
					"3": {
						ID:   3,
						Name: "group3",
					},
				},
				groupMembers: map[string]map[string]gitlab.AccessLevelValue{
					"1": {
						"user2": gitlab.DeveloperPermissions,
					},
					"2": {
						"user1": gitlab.DeveloperPermissions,
						"user3": gitlab.DeveloperPermissions,
					},
					"4": {},
					"3": {
						"user1": gitlab.DeveloperPermissions,
					},
				},
				subgroups: map[string]map[string]struct{}{
					"1": {},
					"2": {
						"4": {},
					},
					"4": {},
					"3": {},
				},
			},
			groupID: "2",
			want: []groupsync.Member{
				&groupsync.GroupMember{
					Grp: &groupsync.Group{
						ID: "4",
						Attributes: &gitlab.Group{
							ID:       4,
							Name:     "sub",
							ParentID: 2,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.GroupMember{
							ID:          2286,
							Username:    "user1",
							Email:       "user1@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &gitlab.GroupMember{
							ID:          3208,
							Username:    "user3",
							Email:       "user3@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
			},
		},
		{
			name: "subgroups_excluded_if_disabled",
			data: &GitLabData{
				users: map[string]*gitlab.User{
					"user1": {
						ID:       2286,
						Username: "user1",
						Email:    "user1@example.com",
					},
					"user2": {
						ID:       5660,
						Username: "user2",
						Email:    "user2@example.com",
					},
					"user3": {
						ID:       3208,
						Username: "user3",
						Email:    "user3@example.com",
					},
				},
				groups: map[string]*gitlab.Group{
					"1": {
						ID:   1,
						Name: "group1",
					},
					"2": {
						ID:   2,
						Name: "group2",
					},
					"4": {
						ID:       4,
						Name:     "sub",
						ParentID: 2,
					},
					"3": {
						ID:   3,
						Name: "group3",
					},
				},
				groupMembers: map[string]map[string]gitlab.AccessLevelValue{
					"1": {
						"user2": gitlab.DeveloperPermissions,
					},
					"2": {
						"user1": gitlab.DeveloperPermissions,
						"user3": gitlab.DeveloperPermissions,
					},
					"4": {},
					"3": {
						"user1": gitlab.DeveloperPermissions,
					},
				},
				subgroups: map[string]map[string]struct{}{
					"1": {},
					"2": {
						"4": {},
					},
					"4": {},
					"3": {},
				},
			},
			opts:    []Opt{WithoutSubGroupsAsMembers()},
			groupID: "2",
			want: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.GroupMember{
							ID:          2286,
							Username:    "user1",
							Email:       "user1@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &gitlab.GroupMember{
							ID:          3208,
							Username:    "user3",
							Email:       "user3@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			server := fakeGitLab(tc.data)
			defer server.Close()

			clientProvider := gitlabClientProvider(server)
			groupRW := NewGroupReadWriter(clientProvider, tc.opts...)

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

func TestGroupReadWriter_GetDescendants(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		data    *GitLabData
		groupID string
		opts    []Opt
		want    []*groupsync.User
		wantErr string
	}{
		{
			name: "success",
			data: &GitLabData{
				users: map[string]*gitlab.User{
					"user1": {
						ID:       2286,
						Username: "user1",
						Email:    "user1@example.com",
					},
					"user2": {
						ID:       5660,
						Username: "user2",
						Email:    "user2@example.com",
					},
					"user3": {
						ID:       3208,
						Username: "user3",
						Email:    "user3@example.com",
					},
				},
				groups: map[string]*gitlab.Group{
					"1": {
						ID:   1,
						Name: "group1",
					},
					"2": {
						ID:   2,
						Name: "group2",
					},
					"3": {
						ID:   3,
						Name: "group3",
					},
				},
				groupMembers: map[string]map[string]gitlab.AccessLevelValue{
					"1": {
						"user2": gitlab.DeveloperPermissions,
					},
					"2": {
						"user1": gitlab.DeveloperPermissions,
						"user3": gitlab.DeveloperPermissions,
					},
					"3": {
						"user1": gitlab.DeveloperPermissions,
					},
				},
				subgroups: map[string]map[string]struct{}{
					"1": {},
					"2": {},
					"3": {},
				},
			},
			groupID: "2",
			want: []*groupsync.User{
				{
					ID: "user1",
					Attributes: &gitlab.GroupMember{
						ID:          2286,
						Username:    "user1",
						Email:       "user1@example.com",
						AccessLevel: gitlab.DeveloperPermissions,
					},
				},
				{
					ID: "user3",
					Attributes: &gitlab.GroupMember{
						ID:          3208,
						Username:    "user3",
						Email:       "user3@example.com",
						AccessLevel: gitlab.DeveloperPermissions,
					},
				},
			},
		},
		{
			name:    "invalid_id",
			data:    &GitLabData{},
			groupID: "invalidID",
			wantErr: "failed to fetch group members for invalidID",
		},
		{
			name: "subgroups_are_included",
			data: &GitLabData{
				users: map[string]*gitlab.User{
					"user1": {
						ID:       2286,
						Username: "user1",
						Email:    "user1@example.com",
					},
					"user2": {
						ID:       5660,
						Username: "user2",
						Email:    "user2@example.com",
					},
					"user3": {
						ID:       3208,
						Username: "user3",
						Email:    "user3@example.com",
					},
				},
				groups: map[string]*gitlab.Group{
					"1": {
						ID:   1,
						Name: "group1",
					},
					"2": {
						ID:   2,
						Name: "group2",
					},
					"4": {
						ID:       4,
						Name:     "sub",
						ParentID: 2,
					},
					"3": {
						ID:   3,
						Name: "group3",
					},
				},
				groupMembers: map[string]map[string]gitlab.AccessLevelValue{
					"1": {
						"user2": gitlab.DeveloperPermissions,
					},
					"2": {
						"user1": gitlab.DeveloperPermissions,
						"user3": gitlab.DeveloperPermissions,
					},
					"4": {
						"user2": gitlab.DeveloperPermissions,
					},
					"3": {
						"user1": gitlab.DeveloperPermissions,
					},
				},
				subgroups: map[string]map[string]struct{}{
					"1": {},
					"2": {
						"4": {},
					},
					"4": {},
					"3": {},
				},
			},
			groupID: "2",
			want: []*groupsync.User{
				{
					ID: "user1",
					Attributes: &gitlab.GroupMember{
						ID:          2286,
						Username:    "user1",
						Email:       "user1@example.com",
						AccessLevel: gitlab.DeveloperPermissions,
					},
				},
				{
					ID: "user2",
					Attributes: &gitlab.GroupMember{
						ID:          5660,
						Username:    "user2",
						Email:       "user2@example.com",
						AccessLevel: gitlab.DeveloperPermissions,
					},
				},
				{
					ID: "user3",
					Attributes: &gitlab.GroupMember{
						ID:          3208,
						Username:    "user3",
						Email:       "user3@example.com",
						AccessLevel: gitlab.DeveloperPermissions,
					},
				},
			},
		},
		{
			name: "subgroups_excluded_if_disabled",
			data: &GitLabData{
				users: map[string]*gitlab.User{
					"user1": {
						ID:       2286,
						Username: "user1",
						Email:    "user1@example.com",
					},
					"user2": {
						ID:       5660,
						Username: "user2",
						Email:    "user2@example.com",
					},
					"user3": {
						ID:       3208,
						Username: "user3",
						Email:    "user3@example.com",
					},
				},
				groups: map[string]*gitlab.Group{
					"1": {
						ID:   1,
						Name: "group1",
					},
					"2": {
						ID:   2,
						Name: "group2",
					},
					"4": {
						ID:       4,
						Name:     "sub",
						ParentID: 2,
					},
					"3": {
						ID:   3,
						Name: "group3",
					},
				},
				groupMembers: map[string]map[string]gitlab.AccessLevelValue{
					"1": {
						"user2": gitlab.DeveloperPermissions,
					},
					"2": {
						"user1": gitlab.DeveloperPermissions,
						"user3": gitlab.DeveloperPermissions,
					},
					"4": {
						"user2": gitlab.DeveloperPermissions,
					},
					"3": {
						"user1": gitlab.DeveloperPermissions,
					},
				},
				subgroups: map[string]map[string]struct{}{
					"1": {},
					"2": {
						"4": {},
					},
					"4": {},
					"3": {},
				},
			},
			opts:    []Opt{WithoutSubGroupsAsMembers()},
			groupID: "2",
			want: []*groupsync.User{
				{
					ID: "user1",
					Attributes: &gitlab.GroupMember{
						ID:          2286,
						Username:    "user1",
						Email:       "user1@example.com",
						AccessLevel: gitlab.DeveloperPermissions,
					},
				},
				{
					ID: "user3",
					Attributes: &gitlab.GroupMember{
						ID:          3208,
						Username:    "user3",
						Email:       "user3@example.com",
						AccessLevel: gitlab.DeveloperPermissions,
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			server := fakeGitLab(tc.data)
			defer server.Close()

			clientProvider := gitlabClientProvider(server)
			groupRW := NewGroupReadWriter(clientProvider, tc.opts...)

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

func TestGroupReadWriter_GetUser(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		data    *GitLabData
		userID  string
		want    *groupsync.User
		wantErr string
	}{
		{
			name: "success",
			data: &GitLabData{
				users: map[string]*gitlab.User{
					"user1": {
						ID:       2286,
						Username: "user1",
						Email:    "user1@example.com",
					},
					"user2": {
						ID:       5660,
						Username: "user2",
						Email:    "user2@example.com",
					},
					"user3": {
						ID:       3208,
						Username: "user3",
						Email:    "user3@example.com",
					},
				},
			},
			userID: "user1",
			want: &groupsync.User{
				ID: "user1",
				Attributes: &gitlab.User{
					ID:       2286,
					Username: "user1",
					Email:    "user1@example.com",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			server := fakeGitLab(tc.data)
			defer server.Close()

			clientProvider := gitlabClientProvider(server)
			groupRW := NewGroupReadWriter(clientProvider)

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

func TestGroupReadWriter_SetMembers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		data         *GitLabData
		opts         []Opt
		groupID      string
		inputMembers []groupsync.Member
		wantMembers  []groupsync.Member
		wantErr      string
	}{
		{
			name: "success_add",
			data: &GitLabData{
				users: map[string]*gitlab.User{
					"user1": {
						ID:       2286,
						Username: "user1",
						Email:    "user1@example.com",
					},
					"user2": {
						ID:       5660,
						Username: "user2",
						Email:    "user2@example.com",
					},
					"user3": {
						ID:       3208,
						Username: "user3",
						Email:    "user3@example.com",
					},
				},
				groups: map[string]*gitlab.Group{
					"1": {
						ID:   1,
						Name: "group1",
					},
					"2": {
						ID:   2,
						Name: "group2",
					},
					"3": {
						ID:   3,
						Name: "group3",
					},
				},
				groupMembers: map[string]map[string]gitlab.AccessLevelValue{
					"1": {
						"user2": gitlab.DeveloperPermissions,
					},
					"2": {
						"user1": gitlab.DeveloperPermissions,
						"user3": gitlab.DeveloperPermissions,
					},
					"3": {
						"user1": gitlab.DeveloperPermissions,
					},
				},
				subgroups: map[string]map[string]struct{}{
					"1": {},
					"2": {},
					"3": {},
				},
			},
			groupID: "1",
			inputMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.User{
							ID:       2286,
							Username: "user1",
							Email:    "user1@example.com",
						},
						Metadata: &AccessLevelMetadata{
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user2",
						Attributes: &gitlab.User{
							ID:       5660,
							Username: "user2",
							Email:    "user2@example.com",
						},
						Metadata: &AccessLevelMetadata{
							AccessLevel: gitlab.MaintainerPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &gitlab.User{
							ID:       3208,
							Username: "user3",
							Email:    "user3@example.com",
						},
						Metadata: &AccessLevelMetadata{
							AccessLevel: gitlab.OwnerPermissions,
						},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.GroupMember{
							ID:          2286,
							Username:    "user1",
							Email:       "user1@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user2",
						Attributes: &gitlab.GroupMember{
							ID:          5660,
							Username:    "user2",
							Email:       "user2@example.com",
							AccessLevel: gitlab.MaintainerPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &gitlab.GroupMember{
							ID:          3208,
							Username:    "user3",
							Email:       "user3@example.com",
							AccessLevel: gitlab.OwnerPermissions,
						},
					},
				},
			},
		},
		{
			name: "success_remove",
			data: &GitLabData{
				users: map[string]*gitlab.User{
					"user1": {
						ID:       2286,
						Username: "user1",
						Email:    "user1@example.com",
					},
					"user2": {
						ID:       5660,
						Username: "user2",
						Email:    "user2@example.com",
					},
					"user3": {
						ID:       3208,
						Username: "user3",
						Email:    "user3@example.com",
					},
				},
				groups: map[string]*gitlab.Group{
					"1": {
						ID:   1,
						Name: "group1",
					},
					"2": {
						ID:   2,
						Name: "group2",
					},
					"3": {
						ID:   3,
						Name: "group3",
					},
				},
				groupMembers: map[string]map[string]gitlab.AccessLevelValue{
					"1": {
						"user2": gitlab.DeveloperPermissions,
					},
					"2": {
						"user1": gitlab.DeveloperPermissions,
						"user3": gitlab.DeveloperPermissions,
					},
					"3": {
						"user1": gitlab.DeveloperPermissions,
					},
				},
				subgroups: map[string]map[string]struct{}{
					"1": {},
					"2": {},
					"3": {},
				},
			},
			groupID: "2",
			inputMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.User{
							ID:       2286,
							Username: "user1",
							Email:    "user1@example.com",
						},
						Metadata: &AccessLevelMetadata{
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.GroupMember{
							ID:          2286,
							Username:    "user1",
							Email:       "user1@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
			},
		},
		{
			name: "success_add_and_remove",
			data: &GitLabData{
				users: map[string]*gitlab.User{
					"user1": {
						ID:       2286,
						Username: "user1",
						Email:    "user1@example.com",
					},
					"user2": {
						ID:       5660,
						Username: "user2",
						Email:    "user2@example.com",
					},
					"user3": {
						ID:       3208,
						Username: "user3",
						Email:    "user3@example.com",
					},
				},
				groups: map[string]*gitlab.Group{
					"1": {
						ID:   1,
						Name: "group1",
					},
					"2": {
						ID:   2,
						Name: "group2",
					},
					"3": {
						ID:   3,
						Name: "group3",
					},
				},
				groupMembers: map[string]map[string]gitlab.AccessLevelValue{
					"1": {
						"user2": gitlab.DeveloperPermissions,
					},
					"2": {
						"user1": gitlab.DeveloperPermissions,
						"user3": gitlab.DeveloperPermissions,
					},
					"3": {
						"user1": gitlab.DeveloperPermissions,
					},
				},
				subgroups: map[string]map[string]struct{}{
					"1": {},
					"2": {},
					"3": {},
				},
			},
			groupID: "2",
			inputMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.User{
							ID:       2286,
							Username: "user1",
							Email:    "user1@example.com",
						},
						Metadata: &AccessLevelMetadata{
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user2",
						Attributes: &gitlab.User{
							ID:       5660,
							Username: "user2",
							Email:    "user2@example.com",
						},
						Metadata: &AccessLevelMetadata{
							AccessLevel: gitlab.OwnerPermissions,
						},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.GroupMember{
							ID:          2286,
							Username:    "user1",
							Email:       "user1@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user2",
						Attributes: &gitlab.GroupMember{
							ID:          5660,
							Username:    "user2",
							Email:       "user2@example.com",
							AccessLevel: gitlab.OwnerPermissions,
						},
					},
				},
			},
		},
		{
			name:    "invalid_id",
			data:    &GitLabData{},
			groupID: "invalidID",
			wantErr: "failed to fetch group members for invalidID",
		},
		{
			name: "success_add_subgroup",
			data: &GitLabData{
				users: map[string]*gitlab.User{
					"user1": {
						ID:       2286,
						Username: "user1",
						Email:    "user1@example.com",
					},
					"user2": {
						ID:       5660,
						Username: "user2",
						Email:    "user2@example.com",
					},
					"user3": {
						ID:       3208,
						Username: "user3",
						Email:    "user3@example.com",
					},
				},
				groups: map[string]*gitlab.Group{
					"1": {
						ID:   1,
						Name: "group1",
					},
					"2": {
						ID:   2,
						Name: "group2",
					},
					"3": {
						ID:   3,
						Name: "group3",
					},
				},
				groupMembers: map[string]map[string]gitlab.AccessLevelValue{
					"1": {
						"user2": gitlab.DeveloperPermissions,
					},
					"2": {
						"user1": gitlab.DeveloperPermissions,
						"user3": gitlab.DeveloperPermissions,
					},
					"3": {
						"user1": gitlab.DeveloperPermissions,
					},
				},
				subgroups: map[string]map[string]struct{}{
					"1": {},
					"2": {},
					"3": {},
				},
			},
			groupID: "2",
			inputMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.User{
							ID:       2286,
							Username: "user1",
							Email:    "user1@example.com",
						},
						Metadata: &AccessLevelMetadata{
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &gitlab.User{
							ID:       3208,
							Username: "user3",
							Email:    "user3@example.com",
						},
						Metadata: &AccessLevelMetadata{
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.GroupMember{
					Grp: &groupsync.Group{
						ID: "3",
						Attributes: &gitlab.Group{
							ID:   3,
							Name: "group3",
						},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.GroupMember{
					Grp: &groupsync.Group{
						ID: "3",
						Attributes: &gitlab.Group{
							ID:       3,
							Name:     "group3",
							ParentID: 2,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.GroupMember{
							ID:          2286,
							Username:    "user1",
							Email:       "user1@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &gitlab.GroupMember{
							ID:          3208,
							Username:    "user3",
							Email:       "user3@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
			},
		},
		{
			name: "success_remove_subgroup",
			data: &GitLabData{
				users: map[string]*gitlab.User{
					"user1": {
						ID:       2286,
						Username: "user1",
						Email:    "user1@example.com",
					},
					"user2": {
						ID:       5660,
						Username: "user2",
						Email:    "user2@example.com",
					},
					"user3": {
						ID:       3208,
						Username: "user3",
						Email:    "user3@example.com",
					},
				},
				groups: map[string]*gitlab.Group{
					"1": {
						ID:   1,
						Name: "group1",
					},
					"2": {
						ID:   2,
						Name: "group2",
					},
					"3": {
						ID:       3,
						Name:     "group3",
						ParentID: 2,
					},
				},
				groupMembers: map[string]map[string]gitlab.AccessLevelValue{
					"1": {
						"user2": gitlab.DeveloperPermissions,
					},
					"2": {
						"user1": gitlab.DeveloperPermissions,
						"user3": gitlab.DeveloperPermissions,
					},
					"3": {
						"user1": gitlab.DeveloperPermissions,
					},
				},
				subgroups: map[string]map[string]struct{}{
					"1": {},
					"2": {"3": {}},
					"3": {},
				},
			},
			groupID: "2",
			inputMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.User{
							ID:       2286,
							Username: "user1",
							Email:    "user1@example.com",
						},
						Metadata: &AccessLevelMetadata{
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &gitlab.User{
							ID:       3208,
							Username: "user3",
							Email:    "user3@example.com",
						},
						Metadata: &AccessLevelMetadata{
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.GroupMember{
							ID:          2286,
							Username:    "user1",
							Email:       "user1@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &gitlab.GroupMember{
							ID:          3208,
							Username:    "user3",
							Email:       "user3@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
			},
		},
		{
			name: "success_add_and_remove_subgroups",
			data: &GitLabData{
				users: map[string]*gitlab.User{
					"user1": {
						ID:       2286,
						Username: "user1",
						Email:    "user1@example.com",
					},
					"user2": {
						ID:       5660,
						Username: "user2",
						Email:    "user2@example.com",
					},
					"user3": {
						ID:       3208,
						Username: "user3",
						Email:    "user3@example.com",
					},
				},
				groups: map[string]*gitlab.Group{
					"1": {
						ID:   1,
						Name: "group1",
					},
					"2": {
						ID:   2,
						Name: "group2",
					},
					"3": {
						ID:       3,
						Name:     "group3",
						ParentID: 2,
					},
					"4": {
						ID:   4,
						Name: "group4",
					},
				},
				groupMembers: map[string]map[string]gitlab.AccessLevelValue{
					"1": {
						"user2": gitlab.DeveloperPermissions,
					},
					"2": {
						"user1": gitlab.DeveloperPermissions,
						"user3": gitlab.DeveloperPermissions,
					},
					"3": {
						"user1": gitlab.DeveloperPermissions,
					},
				},
				subgroups: map[string]map[string]struct{}{
					"1": {},
					"2": {"3": {}},
					"3": {},
				},
			},
			groupID: "2",
			inputMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.User{
							ID:       2286,
							Username: "user1",
							Email:    "user1@example.com",
						},
						Metadata: &AccessLevelMetadata{
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &gitlab.User{
							ID:       3208,
							Username: "user3",
							Email:    "user3@example.com",
						},
						Metadata: &AccessLevelMetadata{
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.GroupMember{
					Grp: &groupsync.Group{
						ID: "4",
						Attributes: &gitlab.Group{
							ID:   4,
							Name: "group4",
						},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.GroupMember{
					Grp: &groupsync.Group{
						ID: "4",
						Attributes: &gitlab.Group{
							ID:       4,
							Name:     "group4",
							ParentID: 2,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.GroupMember{
							ID:          2286,
							Username:    "user1",
							Email:       "user1@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &gitlab.GroupMember{
							ID:          3208,
							Username:    "user3",
							Email:       "user3@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
			},
		},
		{
			name: "subgroups_disabled",
			opts: []Opt{WithoutSubGroupsAsMembers()},
			data: &GitLabData{
				users: map[string]*gitlab.User{
					"user1": {
						ID:       2286,
						Username: "user1",
						Email:    "user1@example.com",
					},
					"user2": {
						ID:       5660,
						Username: "user2",
						Email:    "user2@example.com",
					},
					"user3": {
						ID:       3208,
						Username: "user3",
						Email:    "user3@example.com",
					},
				},
				groups: map[string]*gitlab.Group{
					"1": {
						ID:   1,
						Name: "group1",
					},
					"2": {
						ID:   2,
						Name: "group2",
					},
					"3": {
						ID:   3,
						Name: "group3",
					},
				},
				groupMembers: map[string]map[string]gitlab.AccessLevelValue{
					"1": {
						"user2": gitlab.DeveloperPermissions,
					},
					"2": {
						"user1": gitlab.DeveloperPermissions,
						"user3": gitlab.DeveloperPermissions,
					},
					"3": {
						"user1": gitlab.DeveloperPermissions,
					},
				},
				subgroups: map[string]map[string]struct{}{
					"1": {},
					"2": {},
					"3": {},
				},
			},
			groupID: "2",
			inputMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.User{
							ID:       2286,
							Username: "user1",
							Email:    "user1@example.com",
						},
						Metadata: &AccessLevelMetadata{
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &gitlab.User{
							ID:       3208,
							Username: "user3",
							Email:    "user3@example.com",
						},
						Metadata: &AccessLevelMetadata{
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.GroupMember{
					Grp: &groupsync.Group{
						ID: "3",
						Attributes: &gitlab.Group{
							ID:   3,
							Name: "group3",
						},
					},
				},
			},
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &gitlab.GroupMember{
							ID:          2286,
							Username:    "user1",
							Email:       "user1@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &gitlab.GroupMember{
							ID:          3208,
							Username:    "user3",
							Email:       "user3@example.com",
							AccessLevel: gitlab.DeveloperPermissions,
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			server := fakeGitLab(tc.data)
			defer server.Close()

			clientProvider := gitlabClientProvider(server)
			groupRW := NewGroupReadWriter(clientProvider, tc.opts...)

			err := groupRW.SetMembers(ctx, tc.groupID, tc.inputMembers)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error (-got, +want) = %v", diff)
			}

			gotMembers, err := groupRW.GetMembers(ctx, tc.groupID)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
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

type GitLabData struct {
	users        map[string]*gitlab.User
	groups       map[string]*gitlab.Group
	groupMembers map[string]map[string]gitlab.AccessLevelValue
	subgroups    map[string]map[string]struct{}
}

func (d *GitLabData) findGroupByID(groupID int) *gitlab.Group {
	for _, group := range d.groups {
		if group.ID == groupID {
			return group
		}
	}
	return nil
}

type emptyKeyProvider struct{}

func (p *emptyKeyProvider) Key(ctx context.Context) ([]byte, error) {
	return []byte{}, nil
}

func gitlabClientProvider(server *httptest.Server) *ClientProvider {
	return NewGitLabClientProvider(server.URL, &emptyKeyProvider{}, nil)
}

func fakeGitLab(gitlabData *GitLabData) *httptest.Server {
	mux := http.NewServeMux()
	mux.Handle("GET /api/v4/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := r.FormValue("username")
		user, ok := gitlabData.users[username]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "user not found")
			return
		}
		jsn, err := json.Marshal([]*gitlab.User{user})
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal user")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	mux.Handle("GET /api/v4/groups/{group_id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		groupID := r.PathValue("group_id")
		group, ok := gitlabData.groups[groupID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "group not found")
			return
		}
		jsn, err := json.Marshal(group)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal group")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	mux.Handle("GET /api/v4/groups/{group_id}/members", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		groupID := r.PathValue("group_id")
		members, ok := gitlabData.groupMembers[groupID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "group not found")
			return
		}
		var users []*gitlab.GroupMember
		for username, accessLevel := range members {
			user, ok := gitlabData.users[username]
			if !ok {
				w.WriteHeader(500)
				fmt.Fprintf(w, "user data inconsistency")
				return
			}
			users = append(users, &gitlab.GroupMember{
				ID:          user.ID,
				Username:    user.Username,
				Email:       user.Email,
				AccessLevel: accessLevel,
			})
		}
		jsn, err := json.Marshal(users)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal users")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	mux.Handle("GET /api/v4/groups/{group_id}/subgroups", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		groupID := r.PathValue("group_id")
		members, ok := gitlabData.subgroups[groupID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "group not found")
			return
		}
		var subgroups []*gitlab.Group
		for id := range members {
			subgroup, ok := gitlabData.groups[id]
			if !ok {
				w.WriteHeader(500)
				fmt.Fprintf(w, "user data inconsistency")
				return
			}
			subgroups = append(subgroups, subgroup)
		}
		jsn, err := json.Marshal(subgroups)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal subgroups")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	mux.Handle("POST /api/v4/groups/{group_id}/members", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		groupID := r.PathValue("group_id")
		payload := make(map[string]any)
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to read request body")
			return
		}
		accessLevel, ok := payload["access_level"].(float64)
		if !ok {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to read access level from request")
			return
		}
		username, ok := payload["username"].(string)
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "user not found")
			return
		}
		members, ok := gitlabData.groupMembers[groupID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "group not found")
			return
		}
		user, ok := gitlabData.users[username]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "user not found")
			return
		}
		members[username] = gitlab.AccessLevelValue(accessLevel)
		resp := &gitlab.GroupMember{
			ID:          user.ID,
			Username:    username,
			AccessLevel: gitlab.AccessLevelValue(accessLevel),
		}
		jsn, err := json.Marshal(resp)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal response")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	mux.Handle("PUT /api/v4/groups/{group_id}/members/{user_id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		groupID := r.PathValue("group_id")
		userID, err := strconv.Atoi(r.PathValue("user_id"))
		if err != nil {
			w.WriteHeader(404)
			fmt.Fprintf(w, "user not found")
			return
		}
		var username string
		for _, u := range gitlabData.users {
			if u.ID == userID {
				username = u.Username
				break
			}
		}
		if username == "" {
			w.WriteHeader(404)
			fmt.Fprintf(w, "user not found")
			return
		}
		user, ok := gitlabData.users[username]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "user not found")
			return
		}
		payload := make(map[string]any)
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to read request body")
			return
		}
		accessLevel, ok := payload["access_level"].(float64)
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "access level not found")
			return
		}
		members, ok := gitlabData.groupMembers[groupID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "group not found")
			return
		}
		_, isMember := members[username]
		if !isMember {
			w.WriteHeader(404)
			fmt.Fprintf(w, "user is not member of group")
			return
		}
		members[username] = gitlab.AccessLevelValue(accessLevel)
		resp := &gitlab.GroupMember{
			ID:          user.ID,
			Username:    username,
			AccessLevel: gitlab.AccessLevelValue(accessLevel),
		}
		jsn, err := json.Marshal(resp)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal response")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	mux.Handle("DELETE /api/v4/groups/{group_id}/members/{user_id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		groupID := r.PathValue("group_id")
		userID, err := strconv.Atoi(r.PathValue("user_id"))
		if err != nil {
			w.WriteHeader(404)
			fmt.Fprintf(w, "user not found")
			return
		}
		var username string
		for _, user := range gitlabData.users {
			if user.ID == userID {
				username = user.Username
				break
			}
		}
		if username == "" {
			w.WriteHeader(404)
			fmt.Fprintf(w, "user not found")
			return
		}
		members, ok := gitlabData.groupMembers[groupID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "group not found")
			return
		}
		if _, ok = members[username]; !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "member not found")
			return
		}
		delete(members, username)
		w.WriteHeader(http.StatusNoContent)
	}))
	mux.Handle("POST /api/v4/groups/{id}/transfer", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		groupID, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "missing or malformed group id")
			return
		}
		payload := make(map[string]any)
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to read request body")
			return
		}
		var parentGroupID int
		parentGroupIDPayload, ok := payload["group_id"]
		if ok {
			parentGroupIDFloat, ok := parentGroupIDPayload.(float64)
			if !ok {
				w.WriteHeader(500)
				fmt.Fprintf(w, "malformed parent group id %v", parentGroupIDPayload)
				return
			}
			parentGroupID = int(parentGroupIDFloat)
		}
		childGroup := gitlabData.findGroupByID(groupID)
		if childGroup == nil {
			w.WriteHeader(404)
			fmt.Fprintf(w, "group not found")
			return
		}
		oldParentGroup := gitlabData.findGroupByID(childGroup.ParentID)
		newParentGroup := gitlabData.findGroupByID(parentGroupID)
		childGroup.ParentID = 0
		if newParentGroup != nil {
			childGroup.ParentID = newParentGroup.ID
		}
		if oldParentGroup != nil {
			oldParentSubgroups, ok := gitlabData.subgroups[strconv.Itoa(oldParentGroup.ID)]
			if !ok {
				w.WriteHeader(500)
				fmt.Fprintf(w, "group %d subgroup membership data inconsistent", oldParentGroup.ID)
				return
			}
			delete(oldParentSubgroups, strconv.Itoa(childGroup.ID))
		}
		if newParentGroup != nil {
			newParentSubgroups, ok := gitlabData.subgroups[strconv.Itoa(newParentGroup.ID)]
			if !ok {
				w.WriteHeader(500)
				fmt.Fprintf(w, "group %d subgroup membership data inconsistent", newParentGroup.ID)
				return
			}
			newParentSubgroups[strconv.Itoa(childGroup.ID)] = struct{}{}
		}
		jsn, err := json.Marshal(childGroup)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal group")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	return httptest.NewServer(mux)
}

func sortByID(members []groupsync.Member) {
	slices.SortFunc(members, func(a, b groupsync.Member) int {
		return strings.Compare(a.ID(), b.ID())
	})
}
