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

func TestManyToOneSyncer_Name(t *testing.T) {
	t.Parallel()

	syncer := NewManyToOneSyncer(
		"test syncer",
		"target",
		map[string]GroupReader{},
		&testReadWriteGroupClient{},
		&testOneToOneGroupMapper{},
		&testOneToManyGroupMapper{},
		map[string]UserMapper{},
	)

	res := syncer.Name()
	if res != "test syncer" {
		t.Errorf("unexpected name: %q", res)
	}
}

func TestManyToOneSyncer_Sync(t *testing.T) {
	t.Parallel()

	// key: source user, value: target user
	userMapping1 := map[string]string{
		"su1": "tu1",
		"su2": "tu2",
		"su3": "tu3",
		"su4": "tu4",
	}
	userMapping2 := map[string]string{
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
		userMappers        map[string]UserMapper
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
			userMappers: map[string]UserMapper{
				sourceSystem1: &testUserMapper{
					m: userMapping1,
				},
			},
			syncID: "sg1", // sg1 -> tg1, which mapped to only sg1
			want: map[string][]Member{
				"tg1": {
					&UserMember{Usr: &User{ID: "tu1"}},
					&UserMember{Usr: &User{ID: "tu2"}},
				}, // sg1 members
				"tg2": {},
				"tg3": {},
			},
		},
		{
			name: "sync_empty_group",
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
			userMappers: map[string]UserMapper{
				sourceSystem1: &testUserMapper{
					m: userMapping1,
				},
				sourceSystem2: &testUserMapper{
					m: userMapping2,
				},
			},
			syncID: "sg6", // sg6 -> tg4, which mapped to only sg6
			want: map[string][]Member{
				"tg1": {
					&UserMember{Usr: &User{ID: "tu5"}},
				},
				"tg2": {},
				"tg3": {},
				"tg4": {}, // user removed since sg6 is empty
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
			userMappers: map[string]UserMapper{
				sourceSystem1: &testUserMapper{
					m: userMapping1,
				},
				sourceSystem2: &testUserMapper{
					m: userMapping2,
				},
			},
			syncID: "sg4", // sg4 -> tg3, tg3 -> sg4 (system1) and sg5 (system2)
			want: map[string][]Member{
				"tg1": {},
				"tg2": {},
				"tg3": {
					&UserMember{Usr: &User{ID: "tu1"}}, // from sg4
					&UserMember{Usr: &User{ID: "tu2"}}, // from sg4
					&UserMember{Usr: &User{ID: "tu3"}}, // from sg4
					&UserMember{Usr: &User{ID: "tu4"}}, // from sg4
					&UserMember{Usr: &User{ID: "tu5"}}, // from sg5
				},
			},
		},
		{
			name:               "no_target_ids_found",
			sourceGroupClients: map[string]GroupReader{},
			targetGroupClient:  &testReadWriteGroupClient{},
			sourceGroupMapper:  &testOneToOneGroupMapper{},
			targetGroupMapper:  &testOneToManyGroupMapper{},
			userMappers:        map[string]UserMapper{},
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
				mappedGroupIDsErr: map[string]error{
					"tg1": fmt.Errorf("injected mappedGroupIDsErr for tg1"),
				},
			},
			userMappers: map[string]UserMapper{
				sourceSystem1: &testUserMapper{
					m: userMapping1,
				},
			},
			syncID: "sg1", // sg1 -> tg1
			want: map[string][]Member{
				"tg1": {&UserMember{Usr: &User{ID: "tu5"}}}, // no change to tg1
				"tg2": {},
				"tg3": {},
			},
			wantErr: "injected mappedGroupIDsErr for tg1",
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
					"tg3": {&UserMember{Usr: &User{ID: "tu5"}}},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMappers: map[string]UserMapper{
				sourceSystem1: &testUserMapper{
					m: userMapping1,
				},
			},
			syncID: "sg4", // sg4 -> tg3, tg3 -> sg4 (system1) and sg5 (system2)
			want: map[string][]Member{
				"tg1": {},
				"tg2": {},
				"tg3": {&UserMember{Usr: &User{ID: "tu5"}}}, // no change to tg3
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
					"tg3": {&UserMember{Usr: &User{ID: "tu5"}}},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMappers: map[string]UserMapper{
				sourceSystem1: &testUserMapper{
					m: userMapping1,
				},
				sourceSystem2: &testUserMapper{
					m: userMapping2,
				},
			},
			syncID: "sg4", // sg4 -> tg3, tg3 -> sg4 (system1) and sg5 (system2)
			want: map[string][]Member{
				"tg1": {},
				"tg2": {},
				"tg3": {&UserMember{Usr: &User{ID: "tu5"}}}, // no change to tg3
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
			userMappers: map[string]UserMapper{
				sourceSystem1: &testUserMapper{
					m: userMapping1,
				},
				sourceSystem2: &testUserMapper{
					m: userMapping2,
				},
			},
			syncID: "sg1", // sg1 -> tg1, tg1 -> sg1
			want: map[string][]Member{
				"tg1": {&UserMember{Usr: &User{ID: "tu5"}}}, // no change to tg1
				"tg2": {},
				"tg3": {},
			},
			wantErr: "source group reader not found",
		},
		{
			name: "error_getting_target_user_partial",
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
			userMappers: map[string]UserMapper{
				sourceSystem1: &testUserMapper{
					m: userMapping1,
					mappedUserIDErrs: map[string]error{
						"su1": fmt.Errorf("injected mappedUserIDErrs for su1"),
					},
				},
			},
			syncID: "sg1", // sg1 -> tg1, tg1 -> sg1
			want: map[string][]Member{
				"tg1": {&UserMember{Usr: &User{ID: "tu5"}}}, // no change to tg1
				"tg2": {},
				"tg3": {},
			},
			wantErr: "injected mappedUserIDErrs for su1",
		},
		{
			name: "error_getting_target_users_total",
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
			userMappers: map[string]UserMapper{},
			syncID:      "sg1", // sg1 -> tg1, tg1 -> sg1
			want: map[string][]Member{
				"tg1": {&UserMember{Usr: &User{ID: "tu5"}}}, // no change to tg1
				"tg2": {},
				"tg3": {},
			},
			wantErr: "user mapper not found",
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
					"tg1": {&UserMember{Usr: &User{ID: "tu5"}}},
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
			userMappers: map[string]UserMapper{
				sourceSystem1: &testUserMapper{
					m: userMapping1,
				},
			},
			syncID: "sg1", // sg1 -> tg1, tg1 -> sg1
			want: map[string][]Member{
				"tg1": {&UserMember{Usr: &User{ID: "tu5"}}}, // no change to tg1
				"tg2": {},
				"tg3": {},
			},
			wantErr: "error setting members for group tg1",
		},
		{
			name: "skip_target_user_not_found_error",
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
			userMappers: map[string]UserMapper{
				sourceSystem1: &testUserMapper{
					m: userMapping1,
					mappedUserIDErrs: map[string]error{
						"su1": ErrTargetUserIDNotFound,
					},
				},
			},
			syncID: "sg1", // sg1 -> tg1, tg1 -> sg1
			want: map[string][]Member{
				"tg1": {&UserMember{Usr: &User{ID: "tu2"}}}, // skipped su1 (su1 -> tu1), but added su2 (su2 -> tu2)
				"tg2": {},
				"tg3": {},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			syncer := NewManyToOneSyncer(
				tc.name,
				"target",
				tc.sourceGroupClients,
				tc.targetGroupClient,
				tc.sourceGroupMapper,
				tc.targetGroupMapper,
				tc.userMappers,
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

	userMapping1 := map[string]string{
		"su1": "tu1",
		"su2": "tu2",
		"su3": "tu3",
		"su4": "tu4",
	}
	userMapping2 := map[string]string{
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
		userMappers        map[string]UserMapper
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
			userMappers: map[string]UserMapper{
				sourceSystem1: &testUserMapper{
					m: userMapping1,
				},
				sourceSystem2: &testUserMapper{
					m: userMapping2,
				},
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
			name:               "no_target_ids_found",
			sourceGroupClients: map[string]GroupReader{},
			targetGroupClient:  &testReadWriteGroupClient{},
			sourceGroupMapper:  &testOneToOneGroupMapper{},
			targetGroupMapper: &testOneToManyGroupMapper{
				allGroupIDsErr: fmt.Errorf("injected allGroupIDsErr"),
			},
			userMappers: map[string]UserMapper{},
			wantErr:     "injected allGroupIDsErr",
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
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
				mappedGroupIDsErr: map[string]error{
					"tg1": fmt.Errorf("injected mappedGroupIDErr for tg1"),
				},
			},
			userMappers: map[string]UserMapper{
				sourceSystem1: &testUserMapper{
					m: userMapping1,
				},
				sourceSystem2: &testUserMapper{
					m: userMapping2,
				},
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
			wantErr: "injected mappedGroupIDErr for tg1",
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
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
				mappedGroupIDsErr: map[string]error{
					"tg1": fmt.Errorf("injected mappedGroupIDsErr"),
					"tg2": fmt.Errorf("injected mappedGroupIDsErr"),
					"tg3": fmt.Errorf("injected mappedGroupIDsErr"),
				},
			},
			userMappers: map[string]UserMapper{
				sourceSystem1: &testUserMapper{
					m: userMapping1,
				},
				sourceSystem2: &testUserMapper{
					m: userMapping2,
				},
			},
			want: map[string][]Member{
				"tg1": {},
				"tg2": {},
				"tg3": {},
			},
			wantErr: "injected mappedGroupIDsErr",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			syncer := NewManyToOneSyncer(
				tc.name,
				"target",
				tc.sourceGroupClients,
				tc.targetGroupClient,
				tc.sourceGroupMapper,
				tc.targetGroupMapper,
				tc.userMappers,
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
