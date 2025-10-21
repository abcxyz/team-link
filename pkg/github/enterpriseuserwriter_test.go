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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v67/github"

	"github.com/abcxyz/pkg/testutil"
	"github.com/abcxyz/team-link/v2/pkg/groupsync"
)

func TestEnterpriseUserWriter_SetMembers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                            string
		initialUsers                    map[string]*SCIMUser
		desiredMembers                  []groupsync.Member
		maxUsersToProvision             int64
		failCreateUserCalls             bool
		failListUserCalls               bool
		failUserDeactivationSanityCheck bool
		wantUsersOnServer               map[string]*SCIMUser
		wantErrStr                      string
	}{
		{
			name: "success_create_and_deactivate",
			initialUsers: map[string]*SCIMUser{
				"scim-id-user.old": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-user.old"),
						UserName: "user.old",
						Active:   github.Bool(true),
					},
				},
				"scim-id-user.unchanged": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-user.unchanged"),
						UserName: "user.unchanged",
						Active:   github.Bool(true),
					},
				},
			},
			desiredMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user.new",
						Attributes: &SCIMUser{
							SCIMUserAttributes: github.SCIMUserAttributes{
								UserName: "user.new",
								Active:   github.Bool(true),
							},
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user.unchanged",
						Attributes: &SCIMUser{
							SCIMUserAttributes: github.SCIMUserAttributes{
								UserName: "user.unchanged",
								Active:   github.Bool(true),
							},
						},
					},
				},
			},
			wantUsersOnServer: map[string]*SCIMUser{
				"scim-id-user.old": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
						ID:       github.String("scim-id-user.old"),
						UserName: "user.old",
						Active:   github.Bool(false),
					},
				},
				"scim-id-user.unchanged": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-user.unchanged"),
						UserName: "user.unchanged",
						Active:   github.Bool(true),
					},
				},
				"scim-id-user.new": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-user.new"),
						UserName: "user.new",
						Active:   github.Bool(true),
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
					},
				},
			},
		},
		{
			name:         "success_create_only",
			initialUsers: map[string]*SCIMUser{},
			desiredMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user.one",
						Attributes: &SCIMUser{
							SCIMUserAttributes: github.SCIMUserAttributes{
								UserName: "user.one",
								Active:   github.Bool(true),
							},
						},
					},
				},
			},
			wantUsersOnServer: map[string]*SCIMUser{
				"scim-id-user.one": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-user.one"),
						UserName: "user.one",
						Active:   github.Bool(true),
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
					},
				},
			},
		},
		{
			name: "success_deactivate_only",
			initialUsers: map[string]*SCIMUser{
				"scim-id-user.one": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-user.one"),
						UserName: "user.one",
						Active:   github.Bool(true),
					},
				},
			},
			desiredMembers: []groupsync.Member{},
			wantUsersOnServer: map[string]*SCIMUser{
				"scim-id-user.one": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
						ID:       github.String("scim-id-user.one"),
						UserName: "user.one",
						Active:   github.Bool(false),
					},
				},
			},
		},
		{
			name: "deactivate_sanity_check_fails",
			initialUsers: map[string]*SCIMUser{
				"scim-id-user.one": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-user.one"),
						UserName: "user.one",
						Active:   github.Bool(true),
					},
				},
			},
			desiredMembers:                  []groupsync.Member{},
			failUserDeactivationSanityCheck: true,
			wantUsersOnServer: map[string]*SCIMUser{
				"scim-id-user.one": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-user.one"),
						UserName: "user.one",
						Active:   github.Bool(true),
					},
				},
			},
		},
		{
			name: "success_reactivate_only",
			initialUsers: map[string]*SCIMUser{
				"scim-id-user.one": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-user.one"),
						UserName: "user.one",
						Active:   github.Bool(false),
					},
				},
			},
			desiredMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user.one",
						Attributes: &SCIMUser{
							SCIMUserAttributes: github.SCIMUserAttributes{
								UserName: "user.one",
								Active:   github.Bool(true),
							},
						},
					},
				},
			},
			wantUsersOnServer: map[string]*SCIMUser{
				"scim-id-user.one": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
						ID:       github.String("scim-id-user.one"),
						UserName: "user.one",
						Active:   github.Bool(true),
					},
				},
			},
		},
		{
			name: "no_op",
			initialUsers: map[string]*SCIMUser{
				"scim-id-user.one": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-user.one"),
						UserName: "user.one",
						Active:   github.Bool(false),
					},
				},
			},
			desiredMembers: []groupsync.Member{},
			wantUsersOnServer: map[string]*SCIMUser{
				"scim-id-user.one": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-user.one"),
						UserName: "user.one",
						Active:   github.Bool(false),
					},
				},
			},
		},
		{
			name: "error_list_fails",
			initialUsers: map[string]*SCIMUser{
				"scim-id-user.one": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-user.one"),
						UserName: "user.one",
					},
				},
			},
			failListUserCalls: true,
			wantErrStr:        "failed to list users: failed to list scim users starting at index 1: request failed with status 500",
			wantUsersOnServer: map[string]*SCIMUser{
				"scim-id-user.one": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-user.one"),
						UserName: "user.one",
					},
				},
			},
		},
		{
			name: "error_create_fails",
			initialUsers: map[string]*SCIMUser{
				"scim-id-user.old": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-user.old"),
						UserName: "user.old",
						Active:   github.Bool(true),
					},
				},
			},
			desiredMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID:         "user.new",
						Attributes: &SCIMUser{SCIMUserAttributes: github.SCIMUserAttributes{UserName: "user.new"}},
					},
				},
			},
			failCreateUserCalls: true,
			wantErrStr:          "failed to create \"user.new\"",
			wantUsersOnServer: map[string]*SCIMUser{
				"scim-id-user.old": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
						ID:       github.String("scim-id-user.old"),
						UserName: "user.old",
						Active:   github.Bool(false),
					},
				},
			},
		},
		{
			name:                "error_exceeds_max_users",
			maxUsersToProvision: 1,
			initialUsers:        map[string]*SCIMUser{},
			desiredMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID:         "user.one",
						Attributes: &SCIMUser{SCIMUserAttributes: github.SCIMUserAttributes{UserName: "user.one"}},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID:         "user.two",
						Attributes: &SCIMUser{SCIMUserAttributes: github.SCIMUserAttributes{UserName: "user.two"}},
					},
				},
			},
			wantErrStr: "exceeded max users to provision",
			wantUsersOnServer: map[string]*SCIMUser{
				"scim-id-user.one": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-user.one"),
						UserName: "user.one",
						Active:   github.Bool(true),
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
					},
				},
			},
		},
		{
			name: "chaging roles",
			initialUsers: map[string]*SCIMUser{
				"scim-id-nochange": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-nochange"),
						UserName: "nochange",
						Active:   github.Bool(true),
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
					},
				},
				"scim-id-toadd": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-toadd"),
						UserName: "toadd",
						Active:   github.Bool(true),
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
					},
				},
				"scim-id-toremove": {
					Roles: []*SCIMUserRole{
						{Value: "role_xyz"},
					},
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-toremove"),
						UserName: "toremove",
						Active:   github.Bool(true),
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
					},
				},
				"scim-id-todisable": {
					Roles: []*SCIMUserRole{
						{Value: "role_xyz"},
					},
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-todisable"),
						UserName: "todisable",
						Active:   github.Bool(true),
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
					},
				},
			},
			desiredMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "nochange",
						Attributes: &SCIMUser{
							SCIMUserAttributes: github.SCIMUserAttributes{
								UserName: "nochange",
							},
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "toadd",
						Attributes: &SCIMUser{
							Roles: []*SCIMUserRole{
								{Value: "role_to_add"},
							},
							SCIMUserAttributes: github.SCIMUserAttributes{
								UserName: "toadd",
							},
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "toremove",
						Attributes: &SCIMUser{
							SCIMUserAttributes: github.SCIMUserAttributes{
								UserName: "toremove",
							},
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "tocreate",
						Attributes: &SCIMUser{
							Roles: []*SCIMUserRole{
								{Value: "role_to_create"},
							},
							SCIMUserAttributes: github.SCIMUserAttributes{
								UserName: "tocreate",
							},
						},
					},
				},
			},
			wantUsersOnServer: map[string]*SCIMUser{
				"scim-id-tocreate": {
					Roles: []*SCIMUserRole{
						{Value: "role_to_create"},
					},
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-tocreate"),
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
						UserName: "tocreate",
						Active:   github.Bool(true),
					},
				},
				"scim-id-nochange": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-nochange"),
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
						UserName: "nochange",
						Active:   github.Bool(true),
					},
				},
				"scim-id-toadd": {
					Roles: []*SCIMUserRole{
						{Value: "role_to_add"},
					},
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-toadd"),
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
						UserName: "toadd",
						Active:   github.Bool(true),
					},
				},
				"scim-id-toremove": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-toremove"),
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
						UserName: "toremove",
						Active:   github.Bool(true),
					},
				},
				"scim-id-todisable": {
					SCIMUserAttributes: github.SCIMUserAttributes{
						ID:       github.String("scim-id-todisable"),
						Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
						UserName: "todisable",
						Active:   github.Bool(false),
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			userData := &EnterpriseUserData{
				allUsers:            tc.initialUsers,
				failCreateUserCalls: tc.failCreateUserCalls,
				failListUserCalls:   tc.failListUserCalls,
			}
			srv := fakeEnterprise(t, userData)
			defer srv.Close()

			var opts []EnterpriseRWOpt
			if tc.maxUsersToProvision > 0 {
				opts = append(opts, WithMaxUsersToProvision(tc.maxUsersToProvision))
			}

			if tc.failUserDeactivationSanityCheck {
				opts = append(opts, WithUserDeactivationSanityCheck(func(context.Context, *groupsync.User) (bool, error) {
					return false, nil
				}))
			}

			writer, err := NewEnterpriseUserWriter(srv.Client(), srv.URL, opts...)
			if err != nil {
				t.Fatalf("NewEnterpriseUserWriter failed: %v", err)
			}

			err = writer.SetMembers(ctx, "test-group", tc.desiredMembers)
			if diff := testutil.DiffErrString(err, tc.wantErrStr); diff != "" {
				t.Errorf("error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantUsersOnServer, userData.allUsers); diff != "" {
				t.Errorf("users on server mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
