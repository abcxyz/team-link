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

package groupsync

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abcxyz/pkg/testutil"
)

func TestManyToOneSyncer_Sync(t *testing.T) {
	t.Parallel()

	// key: source user, value: target user
	userMapping := map[string]string{
		"su1": "tu1",
		"su2": "tu2",
		"su3": "tu3",
		"su4": "tu4",
		"su5": "tu5",
	}

	sourceSystem1 := "ss1"
	sourceGroups1 := map[string]*Group{
		"sg1": {ID: "sg1"},
		"sg2": {ID: "sg2"},
		"sg3": {ID: "sg3"},
		"sg4": {ID: "sg4"},
	}
	sourceUsers1 := map[string]*User{
		"su1": {ID: "su1"},
		"su2": {ID: "su2"},
		"su3": {ID: "su3"},
		"su4": {ID: "su4"},
	}
	sourceGroupMembers1 := map[string][]Member{
		"sg1": {
			&UserMember{Usr: &User{ID: "su1"}},
			&GroupMember{Grp: &Group{ID: "sg2"}},
		},
		"sg2": {
			&UserMember{Usr: &User{ID: "su2"}},
		},
		"sg3": {
			&UserMember{Usr: &User{ID: "su3"}},
			&UserMember{Usr: &User{ID: "su4"}},
		},
		"sg4": {
			&GroupMember{Grp: &Group{ID: "sg1"}}, // nested group sg1
			&GroupMember{Grp: &Group{ID: "sg3"}},
		},
	}

	sourceSystem2 := "ss2"
	sourceGroups2 := map[string]*Group{
		"sg5": {ID: "sg5"},
		"sg6": {ID: "sg6"},
	}
	sourceUsers2 := map[string]*User{
		"su5": {ID: "su5"},
	}
	sourceGroupMembers2 := map[string][]Member{
		"sg5": {
			&UserMember{Usr: &User{ID: "su5"}},
		},
		"sg6": {},
	}

	targetGroups := map[string]*Group{
		"tg1": {ID: "tg1"},
		"tg2": {ID: "tg2"},
		"tg3": {ID: "tg3"},
		"tg4": {ID: "tg4"},
	}
	targetUsers := map[string]*User{
		"tu1": {ID: "tu1"},
		"tu2": {ID: "tu2"},
		"tu3": {ID: "tu3"},
		"tu4": {ID: "tu4"},
		"tu5": {ID: "tu5"},
	}

	// One to one source group to target group mappings.
	sourceGroupMapping := map[string]Mapping{
		"sg1": {GroupID: "tg1"},
		"sg2": {GroupID: "tg2"},
		"sg3": {GroupID: "tg2"},
		"sg4": {GroupID: "tg3"},
		"sg5": {GroupID: "tg3"},
		"sg6": {GroupID: "tg4"},
	}
	// One to many target group to source groups mappings.
	targetGroupMapping := map[string][]Mapping{
		"tg1": {{GroupID: "sg1", System: sourceSystem1}},
		"tg2": {
			{GroupID: "sg2", System: sourceSystem1},
			{GroupID: "sg3", System: sourceSystem1},
		},
		"tg3": {
			{GroupID: "sg4", System: sourceSystem1},
			{GroupID: "sg5", System: sourceSystem2},
		},
		"tg4": {
			{GroupID: "sg6", System: sourceSystem2},
		},
	}

	cases := []struct {
		name               string
		sourceGroupClients map[string]GroupReader
		targetGroupClient  GroupReadWriter
		sourceGroupMapper  OneToOneGroupMapper
		targetGroupMapper  OneToManyGroupMapper
		userMapper         UserMapper
		syncID             string
		want               map[string][]Member
		wantErr            string
	}{
		{
			name: "single_source",
			sourceGroupClients: map[string]GroupReader{
				sourceSystem1: &testReadWriteGroupClient{
					groups:       sourceGroups1,
					groupMembers: sourceGroupMembers1,
					users:        sourceUsers1,
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				// groupMembers is subject to change after sync is completed.
				groupMembers: map[string][]Member{
					"tg1": {&UserMember{Usr: &User{ID: "tu5"}}},
					"tg2": {},
					"tg3": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: userMapping,
			},
			syncID: "sg1",
			want: map[string][]Member{
				"tg1": {
					&UserMember{Usr: &User{ID: "tu1"}},
					&UserMember{Usr: &User{ID: "tu2"}},
				},
				"tg2": {},
				"tg3": {},
			},
		},
		{
			name: "empty_group",
			sourceGroupClients: map[string]GroupReader{
				sourceSystem1: &testReadWriteGroupClient{
					groups:       sourceGroups1,
					groupMembers: sourceGroupMembers1,
					users:        sourceUsers1,
				},
				sourceSystem2: &testReadWriteGroupClient{
					groups:       sourceGroups2,
					groupMembers: sourceGroupMembers2,
					users:        sourceUsers2,
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				// groupMembers is subject to change after sync is completed.
				groupMembers: map[string][]Member{
					"tg1": {&UserMember{Usr: &User{ID: "tu5"}}},
					"tg2": {},
					"tg3": {},
					"tg4": {&UserMember{Usr: &User{ID: "tu5"}}},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: userMapping,
			},
			syncID: "sg6",
			want: map[string][]Member{
				"tg1": {
					&UserMember{Usr: &User{ID: "tu5"}},
				},
				"tg2": {},
				"tg3": {},
				"tg4": {},
			},
		},
		{
			name: "multiple_sources",
			sourceGroupClients: map[string]GroupReader{
				sourceSystem1: &testReadWriteGroupClient{
					groups:       sourceGroups1,
					groupMembers: sourceGroupMembers1,
					users:        sourceUsers1,
				},
				sourceSystem2: &testReadWriteGroupClient{
					groups:       sourceGroups2,
					groupMembers: sourceGroupMembers2,
					users:        sourceUsers2,
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				// groupMembers is subject to change after sync is completed.
				groupMembers: map[string][]Member{
					"tg1": {},
					"tg2": {},
					"tg3": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: userMapping,
			},
			syncID: "sg4",
			want: map[string][]Member{
				"tg1": {},
				"tg2": {},
				"tg3": {
					&UserMember{Usr: &User{ID: "tu1"}},
					&UserMember{Usr: &User{ID: "tu2"}},
					&UserMember{Usr: &User{ID: "tu3"}},
					&UserMember{Usr: &User{ID: "tu4"}},
					&UserMember{Usr: &User{ID: "tu5"}},
				},
			},
		},
		{
			name:               "no_target_ids_found",
			sourceGroupClients: map[string]GroupReader{},
			targetGroupClient:  &testReadWriteGroupClient{},
			sourceGroupMapper:  &testOneToOneGroupMapper{},
			targetGroupMapper:  &testOneToManyGroupMapper{},
			userMapper:         &testUserMapper{},
			syncID:             "sg1",
			wantErr:            "error fetching target group id",
		},
		{
			name: "error_getting_associated_source_ids",
			sourceGroupClients: map[string]GroupReader{
				sourceSystem1: &testReadWriteGroupClient{
					groups:       sourceGroups1,
					groupMembers: sourceGroupMembers1,
					users:        sourceUsers1,
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				// groupMembers is subject to change after sync is completed.
				groupMembers: map[string][]Member{
					"tg1": {},
					"tg2": {},
					"tg3": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
				mappedGroupIdsErr: map[string]error{
					"tg1": fmt.Errorf("injected mappedGroupIdsErr for tg1"),
				},
			},
			userMapper: &testUserMapper{
				m: userMapping,
			},
			syncID: "sg1",
			want: map[string][]Member{
				"tg1": {},
				"tg2": {},
				"tg3": {},
			},
			wantErr: "injected mappedGroupIdsErr for tg1",
		},
		{
			name: "error_getting_source_users_partial_system_not_found",
			sourceGroupClients: map[string]GroupReader{
				sourceSystem1: &testReadWriteGroupClient{
					groups:       sourceGroups1,
					groupMembers: sourceGroupMembers1,
					users:        sourceUsers1,
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				// groupMembers is subject to change after sync is completed.
				groupMembers: map[string][]Member{
					"tg1": {},
					"tg2": {},
					"tg3": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: userMapping,
			},
			syncID: "sg4",
			want: map[string][]Member{
				"tg1": {},
				"tg2": {},
				"tg3": {
					&UserMember{Usr: &User{ID: "tu1"}},
					&UserMember{Usr: &User{ID: "tu2"}},
					&UserMember{Usr: &User{ID: "tu3"}},
					&UserMember{Usr: &User{ID: "tu4"}},
				},
			},
			wantErr: "source group reader not found",
		},
		{
			name: "error_getting_source_users_partial_descendants_err",
			sourceGroupClients: map[string]GroupReader{
				sourceSystem1: &testReadWriteGroupClient{
					groups:       sourceGroups1,
					groupMembers: sourceGroupMembers1,
					users:        sourceUsers1,
				},
				sourceSystem2: &testReadWriteGroupClient{
					groups:       sourceGroups2,
					groupMembers: sourceGroupMembers2,
					users:        sourceUsers2,
					descendantsErrs: map[string]error{
						"sg5": fmt.Errorf("injected descendantsErrs for sg5"),
					},
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				// groupMembers is subject to change after sync is completed.
				groupMembers: map[string][]Member{
					"tg1": {},
					"tg2": {},
					"tg3": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: userMapping,
			},
			syncID: "sg4",
			want: map[string][]Member{
				"tg1": {},
				"tg2": {},
				"tg3": {
					&UserMember{Usr: &User{ID: "tu1"}},
					&UserMember{Usr: &User{ID: "tu2"}},
					&UserMember{Usr: &User{ID: "tu3"}},
					&UserMember{Usr: &User{ID: "tu4"}},
				},
			},
			wantErr: "injected descendantsErrs for sg5",
		},
		{
			name:               "error_getting_source_users_total",
			sourceGroupClients: map[string]GroupReader{},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				// groupMembers is subject to change after sync is completed.
				groupMembers: map[string][]Member{
					"tg1": {},
					"tg2": {},
					"tg3": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: userMapping,
			},
			syncID: "sg1",
			want: map[string][]Member{
				"tg1": {},
				"tg2": {},
				"tg3": {},
			},
			wantErr: "source group reader not found",
		},
		{
			name: "error_mapping_source_users_partial",
			sourceGroupClients: map[string]GroupReader{
				sourceSystem1: &testReadWriteGroupClient{
					groups:       sourceGroups1,
					groupMembers: sourceGroupMembers1,
					users:        sourceUsers1,
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				// groupMembers is subject to change after sync is completed.
				groupMembers: map[string][]Member{
					"tg1": {},
					"tg2": {},
					"tg3": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: userMapping,
				mappedUserIDErrs: map[string]error{
					"su1": fmt.Errorf("injected mappedUserIDErrs for su1"),
				},
			},
			syncID: "sg1",
			want: map[string][]Member{
				"tg1": {
					&UserMember{Usr: &User{ID: "tu2"}},
				},
				"tg2": {},
				"tg3": {},
			},
			wantErr: "injected mappedUserIDErrs for su1",
		},
		{
			name: "error_mapping_source_users_total",
			sourceGroupClients: map[string]GroupReader{
				sourceSystem1: &testReadWriteGroupClient{
					groups:       sourceGroups1,
					groupMembers: sourceGroupMembers1,
					users:        sourceUsers1,
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				// groupMembers is subject to change after sync is completed.
				groupMembers: map[string][]Member{
					"tg1": {},
					"tg2": {},
					"tg3": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				mappedUserIDErrs: map[string]error{
					"su1": fmt.Errorf("injected mappedUserIDErrs"),
					"su2": fmt.Errorf("injected mappedUserIDErrs"),
				},
			},
			syncID: "sg1",
			want: map[string][]Member{
				"tg1": {},
				"tg2": {},
				"tg3": {},
			},
			wantErr: "injected mappedUserIDErrs",
		},
		{
			name: "error_setting_members",
			sourceGroupClients: map[string]GroupReader{
				sourceSystem1: &testReadWriteGroupClient{
					groups:       sourceGroups1,
					groupMembers: sourceGroupMembers1,
					users:        sourceUsers1,
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				// groupMembers is subject to change after sync is completed.
				groupMembers: map[string][]Member{
					"tg1": {},
					"tg2": {},
					"tg3": {},
				},
				setMembersErrs: map[string]error{
					"tg1": fmt.Errorf("error setting members for group tg1"),
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: userMapping,
			},
			syncID: "sg1",
			want: map[string][]Member{
				"tg1": {},
				"tg2": {},
				"tg3": {},
			},
			wantErr: "error setting members for group tg1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			syncer := NewManyToOneSyncer(
				"target",
				tc.sourceGroupClients,
				tc.targetGroupClient,
				tc.sourceGroupMapper,
				tc.targetGroupMapper,
				tc.userMapper,
			)

			err := syncer.Sync(ctx, tc.syncID)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error (-want, +got):\n%s", diff)
			}
			for targetGroupID := range tc.want {
				got, err := tc.targetGroupClient.GetMembers(ctx, targetGroupID)
				if err != nil {
					// test data misconfigured. fail fast
					t.Fatalf("test data misconfigured. failed to get target group members: %v", err)
				}
				if diff := cmp.Diff(got, tc.want[targetGroupID]); diff != "" {
					t.Errorf("unexpected result for targetGroupID %s (-got, +want): \n%s", targetGroupID, diff)
				}
			}
		})
	}
}

func TestManyToOneSyncer_SyncAll(t *testing.T) {
	t.Parallel()

	// key: source user, value: target user
	userMapping := map[string]string{
		"su1": "tu1",
		"su2": "tu2",
		"su3": "tu3",
		"su4": "tu4",
		"su5": "tu5",
	}

	sourceSystem1 := "ss1"
	sourceGroups1 := map[string]*Group{
		"sg1": {ID: "sg1"},
		"sg2": {ID: "sg2"},
		"sg3": {ID: "sg3"},
		"sg4": {ID: "sg4"},
	}
	sourceUsers1 := map[string]*User{
		"su1": {ID: "su1"},
		"su2": {ID: "su2"},
		"su3": {ID: "su3"},
		"su4": {ID: "su4"},
	}
	sourceGroupMembers1 := map[string][]Member{
		"sg1": {
			&UserMember{Usr: &User{ID: "su1"}},
			&GroupMember{Grp: &Group{ID: "sg2"}},
		},
		"sg2": {
			&UserMember{Usr: &User{ID: "su2"}},
		},
		"sg3": {
			&UserMember{Usr: &User{ID: "su3"}},
			&UserMember{Usr: &User{ID: "su4"}},
		},
		"sg4": {
			&GroupMember{Grp: &Group{ID: "sg1"}}, // nested group sg1
			&GroupMember{Grp: &Group{ID: "sg3"}},
		},
	}

	sourceSystem2 := "ss2"
	sourceGroups2 := map[string]*Group{
		"sg5": {ID: "sg5"},
	}
	sourceUsers2 := map[string]*User{
		"su5": {ID: "su5"},
	}
	sourceGroupMembers2 := map[string][]Member{
		"sg5": {
			&UserMember{Usr: &User{ID: "su5"}},
		},
	}

	targetGroups := map[string]*Group{
		"tg1": {ID: "tg1"},
		"tg2": {ID: "tg2"},
		"tg3": {ID: "tg3"},
	}
	targetUsers := map[string]*User{
		"tu1": {ID: "tu1"},
		"tu2": {ID: "tu2"},
		"tu3": {ID: "tu3"},
		"tu4": {ID: "tu4"},
		"tu5": {ID: "tu5"},
	}

	// One to one source group to target group mappings.
	sourceGroupMapping := map[string]Mapping{
		"sg1": {GroupID: "tg1"},
		"sg2": {GroupID: "tg2"},
		"sg3": {GroupID: "tg2"},
		"sg4": {GroupID: "tg3"},
		"sg5": {GroupID: "tg3"},
	}
	// One to many target group to source groups mappings.
	targetGroupMapping := map[string][]Mapping{
		"tg1": {{GroupID: "sg1", System: sourceSystem1}},
		"tg2": {
			{GroupID: "sg2", System: sourceSystem1},
			{GroupID: "sg3", System: sourceSystem1},
		},
		"tg3": {
			{GroupID: "sg4", System: sourceSystem1},
			{GroupID: "sg5", System: sourceSystem2},
		},
	}

	cases := []struct {
		name               string
		sourceGroupClients map[string]GroupReader
		targetGroupClient  GroupReadWriter
		sourceGroupMapper  OneToOneGroupMapper
		targetGroupMapper  OneToManyGroupMapper
		userMapper         UserMapper
		want               map[string][]Member
		wantErr            string
	}{
		{
			name: "sync_all_success",

			sourceGroupClients: map[string]GroupReader{
				sourceSystem1: &testReadWriteGroupClient{
					groups:       sourceGroups1,
					groupMembers: sourceGroupMembers1,
					users:        sourceUsers1,
				},
				sourceSystem2: &testReadWriteGroupClient{
					groups:       sourceGroups2,
					groupMembers: sourceGroupMembers2,
					users:        sourceUsers2,
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				// groupMembers is subject to change after sync is completed.
				groupMembers: map[string][]Member{
					"tg1": {},
					"tg2": {},
					"tg3": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: userMapping,
			},
			want: map[string][]Member{
				"tg1": {
					&UserMember{Usr: &User{ID: "tu1"}},
					&UserMember{Usr: &User{ID: "tu2"}},
				},
				"tg2": {
					&UserMember{Usr: &User{ID: "tu2"}},
					&UserMember{Usr: &User{ID: "tu3"}},
					&UserMember{Usr: &User{ID: "tu4"}},
				},
				"tg3": {
					&UserMember{Usr: &User{ID: "tu1"}},
					&UserMember{Usr: &User{ID: "tu2"}},
					&UserMember{Usr: &User{ID: "tu3"}},
					&UserMember{Usr: &User{ID: "tu4"}},
					&UserMember{Usr: &User{ID: "tu5"}},
				},
			},
		},
		{
			name:               "no_source_ids_found",
			sourceGroupClients: map[string]GroupReader{},
			targetGroupClient:  &testReadWriteGroupClient{},
			sourceGroupMapper: &testOneToOneGroupMapper{
				allGroupIDsErr: fmt.Errorf("injected allGroupIDsErr"),
			},
			targetGroupMapper: &testOneToManyGroupMapper{},
			userMapper:        &testUserMapper{},
			wantErr:           "injected allGroupIDsErr",
		},
		{
			name: "sync_all_partial_failure",
			sourceGroupClients: map[string]GroupReader{
				sourceSystem1: &testReadWriteGroupClient{
					groups:       sourceGroups1,
					groupMembers: sourceGroupMembers1,
					users:        sourceUsers1,
				},
				sourceSystem2: &testReadWriteGroupClient{
					groups:       sourceGroups2,
					groupMembers: sourceGroupMembers2,
					users:        sourceUsers2,
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				// groupMembers is subject to change after sync is completed.
				groupMembers: map[string][]Member{
					"tg1": {},
					"tg2": {},
					"tg3": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
				mappedGroupIdErr: map[string]error{
					"sg1": fmt.Errorf("injected mappedGroupIdErr for sg1"),
				},
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: userMapping,
			},
			want: map[string][]Member{
				"tg1": {},
				"tg2": {
					&UserMember{Usr: &User{ID: "tu2"}},
					&UserMember{Usr: &User{ID: "tu3"}},
					&UserMember{Usr: &User{ID: "tu4"}},
				},
				"tg3": {
					&UserMember{Usr: &User{ID: "tu1"}},
					&UserMember{Usr: &User{ID: "tu2"}},
					&UserMember{Usr: &User{ID: "tu3"}},
					&UserMember{Usr: &User{ID: "tu4"}},
					&UserMember{Usr: &User{ID: "tu5"}},
				},
			},
			wantErr: "injected mappedGroupIdErr for sg1",
		},
		{
			name: "sync_all_total_failure",
			sourceGroupClients: map[string]GroupReader{
				sourceSystem1: &testReadWriteGroupClient{
					groups:       sourceGroups1,
					groupMembers: sourceGroupMembers1,
					users:        sourceUsers1,
				},
				sourceSystem2: &testReadWriteGroupClient{
					groups:       sourceGroups2,
					groupMembers: sourceGroupMembers2,
					users:        sourceUsers2,
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				// groupMembers is subject to change after sync is completed.
				groupMembers: map[string][]Member{
					"tg1": {},
					"tg2": {},
					"tg3": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
				mappedGroupIdErr: map[string]error{
					"sg1": fmt.Errorf("injected mappedGroupIdsErr"),
					"sg2": fmt.Errorf("injected mappedGroupIdsErr"),
					"sg3": fmt.Errorf("injected mappedGroupIdsErr"),
					"sg4": fmt.Errorf("injected mappedGroupIdsErr"),
					"sg5": fmt.Errorf("injected mappedGroupIdsErr"),
				},
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: userMapping,
			},
			want: map[string][]Member{
				"tg1": {},
				"tg2": {},
				"tg3": {},
			},
			wantErr: "injected mappedGroupIdsErr",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			syncer := NewManyToOneSyncer(
				"target",
				tc.sourceGroupClients,
				tc.targetGroupClient,
				tc.sourceGroupMapper,
				tc.targetGroupMapper,
				tc.userMapper,
			)

			err := syncer.SyncAll(ctx)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error (-want, +got):\n%s", diff)
			}
			for targetGroupID := range tc.want {
				got, err := tc.targetGroupClient.GetMembers(ctx, targetGroupID)
				if err != nil {
					// test data misconfigured. fail fast
					t.Fatalf("test data misconfigured. failed to get target group members: %v", err)
				}
				if diff := cmp.Diff(got, tc.want[targetGroupID]); diff != "" {
					t.Errorf("unexpected result for targetGroupID %s (-got, +want): \n%s", targetGroupID, diff)
				}
			}
		})
	}
}
