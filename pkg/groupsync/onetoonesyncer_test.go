// Copyright 2025 Google LLC
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

func TestOneToOneSyncer_NewOneToOneSyncer(t *testing.T) {
	t.Parallel()

	params := &OneToOneSyncerParams{
		Name:              "test_syncer",
		SourceSystem:      "source",
		TargetSystem:      "target",
		SourceGroupReader: &testReadWriteGroupClient{},
		TargetGroupWriter: &testReadWriteGroupClient{},
		SourceGroupMapper: &testOneToOneGroupMapper{},
		UserMapper:        &testUserMapper{},
	}
	syncer := NewOneToOneSyncer(params)

	if got, want := syncer.Name(), "test_syncer"; got != want {
		t.Errorf("unexpected name: got %q, want %q", got, want)
	}
	if got, want := syncer.SourceSystem(), "source"; got != want {
		t.Errorf("unexpected source system: got %q, want %q", got, want)
	}
	if got, want := syncer.TargetSystem(), "target"; got != want {
		t.Errorf("unexpected target system: got %q, want %q", got, want)
	}
}

func TestOneToOneSyncer_Sync(t *testing.T) {
	t.Parallel()

	// key: source user, value: target user
	userMapping := map[string]string{
		"su1": "tu1",
		"su2": "tu2",
	}

	sourceGroups := map[string]*Group{
		"sg1": {ID: "sg1"},
		"sg2": {ID: "sg2"},
	}
	sourceUsers := map[string]*User{
		"su1": {ID: "su1"},
		"su2": {ID: "su2"},
	}
	sourceGroupMembers := map[string][]Member{
		"sg1": {
			&UserMember{Usr: &User{ID: "su1"}},
			&GroupMember{Grp: &Group{ID: "sg2"}},
		},
		"sg2": {
			&UserMember{Usr: &User{ID: "su2"}},
		},
	}

	targetGroups := map[string]*Group{
		"tg1": {ID: "tg1"},
		"tg2": {ID: "tg2"},
	}
	targetUsers := map[string]*User{
		"tu1": {ID: "tu1"},
		"tu2": {ID: "tu2"},
	}

	// One to one source group to target group mappings.
	sourceGroupMapping := map[string]Mapping{
		"sg1": {GroupID: "tg1"},
		"sg2": {GroupID: "tg2"},
	}

	cases := []struct {
		name              string
		sourceGroupClient GroupReader
		targetGroupClient GroupReadWriter
		sourceGroupMapper OneToOneGroupMapper
		userMapper        UserMapper
		syncID            string
		want              map[string][]Member
		wantErr           string
	}{
		{
			name: "success",
			sourceGroupClient: &testReadWriteGroupClient{
				groups:       sourceGroups,
				groupMembers: sourceGroupMembers,
				users:        sourceUsers,
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				groupMembers: map[string][]Member{
					"tg1": {},
					"tg2": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
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
			},
		},
		{
			name: "err_getting_target_group_id",
			sourceGroupMapper: &testOneToOneGroupMapper{
				m:                sourceGroupMapping,
				mappedGroupIDErr: map[string]error{"sg1": fmt.Errorf("injected mapped group id error")},
			},
			syncID:  "sg1",
			wantErr: "injected mapped group id error",
		},
		{
			name: "err_getting_source_users",
			sourceGroupClient: &testReadWriteGroupClient{
				descendantsErrs: map[string]error{"sg1": fmt.Errorf("injected descendants error")},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			syncID:  "sg1",
			wantErr: "injected descendants error",
		},
		{
			name: "err_mapping_users",
			sourceGroupClient: &testReadWriteGroupClient{
				groups:       sourceGroups,
				groupMembers: sourceGroupMembers,
				users:        sourceUsers,
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				groupMembers: map[string][]Member{
					"tg1": {&UserMember{Usr: &User{ID: "tu1"}}},
					"tg2": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			userMapper: &testUserMapper{
				m:                userMapping,
				mappedUserIDErrs: map[string]error{"su1": fmt.Errorf("injected mapped user id error")},
			},
			syncID: "sg1", // -> tg1
			want: map[string][]Member{
				"tg1": {&UserMember{Usr: &User{ID: "tu1"}}}, // no change
			},
			wantErr: "injected mapped user id error",
		},
		{
			name: "skipp_target_user_not_found_err",
			sourceGroupClient: &testReadWriteGroupClient{
				groups:       sourceGroups,
				groupMembers: sourceGroupMembers,
				users:        sourceUsers,
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				groupMembers: map[string][]Member{
					"tg1": {&UserMember{Usr: &User{ID: "tu1"}}},
					"tg2": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			userMapper: &testUserMapper{
				m:                userMapping,
				mappedUserIDErrs: map[string]error{"su1": ErrTargetUserIDNotFound},
			},
			syncID: "sg1", // -> tg1
			want: map[string][]Member{
				"tg1": {&UserMember{Usr: &User{ID: "tu2"}}}, // skipped su1 (tu1), but added su2 (tu2)
			},
		},
		{
			name: "err_setting_members",
			sourceGroupClient: &testReadWriteGroupClient{
				groups:       sourceGroups,
				groupMembers: sourceGroupMembers,
				users:        sourceUsers,
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				groupMembers: map[string][]Member{
					"tg1": {&UserMember{Usr: &User{ID: "tu1"}}},
					"tg2": {},
				},
				setMembersErrs: map[string]error{"tg1": fmt.Errorf("injected set members error")},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
			},
			userMapper: &testUserMapper{
				m: userMapping,
			},
			syncID: "sg1", // -> tg1
			want: map[string][]Member{
				"tg1": {&UserMember{Usr: &User{ID: "tu1"}}}, // no change
			},
			wantErr: "injected set members error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			syncer := NewOneToOneSyncer(&OneToOneSyncerParams{
				Name:              tc.name,
				SourceSystem:      "source",
				TargetSystem:      "target",
				SourceGroupReader: tc.sourceGroupClient,
				TargetGroupWriter: tc.targetGroupClient,
				SourceGroupMapper: tc.sourceGroupMapper,
				UserMapper:        tc.userMapper,
			})

			err := syncer.Sync(ctx, tc.syncID)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error diff: %v", tc.name, diff)
			}

			for targetGroupID, members := range tc.want {
				got, err := tc.targetGroupClient.GetMembers(ctx, targetGroupID)
				if err != nil {
					t.Fatalf("failed to get target group members for group %s: %v", targetGroupID, err)
				}
				if diff := cmp.Diff(members, got, cmp.AllowUnexported(User{}, Group{})); diff != "" {
					t.Errorf("Process(%+v) got diff (-want, +got): %v", tc.name, diff)
				}
			}
		})
	}
}

func TestOneToOneSyncer_SyncAll(t *testing.T) {
	t.Parallel()

	// key: source user, value: target user
	userMapping := map[string]string{
		"su1": "tu1",
		"su2": "tu2",
	}

	sourceGroups := map[string]*Group{
		"sg1": {ID: "sg1"},
		"sg2": {ID: "sg2"},
	}
	sourceUsers := map[string]*User{
		"su1": {ID: "su1"},
		"su2": {ID: "su2"},
	}
	sourceGroupMembers := map[string][]Member{
		"sg1": {
			&UserMember{Usr: &User{ID: "su1"}},
			&GroupMember{Grp: &Group{ID: "sg2"}},
		},
		"sg2": {
			&UserMember{Usr: &User{ID: "su2"}},
		},
	}

	targetGroups := map[string]*Group{
		"tg1": {ID: "tg1"},
		"tg2": {ID: "tg2"},
	}
	targetUsers := map[string]*User{
		"tu1": {ID: "tu1"},
		"tu2": {ID: "tu2"},
	}

	// One to one source group to target group mappings.
	sourceGroupMapping := map[string]Mapping{
		"sg1": {GroupID: "tg1"},
		"sg2": {GroupID: "tg2"},
	}

	cases := []struct {
		name              string
		sourceGroupClient GroupReader
		targetGroupClient GroupReadWriter
		sourceGroupMapper OneToOneGroupMapper
		userMapper        UserMapper
		want              map[string][]Member
		wantErr           string
	}{
		{
			name: "success",
			sourceGroupClient: &testReadWriteGroupClient{
				groups:       sourceGroups,
				groupMembers: sourceGroupMembers,
				users:        sourceUsers,
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				groupMembers: map[string][]Member{
					"tg1": {},
					"tg2": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m: sourceGroupMapping,
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
				},
			},
		},
		{
			name: "err_getting_all_group_ids",
			sourceGroupMapper: &testOneToOneGroupMapper{
				m:              sourceGroupMapping,
				allGroupIDsErr: fmt.Errorf("injected all group ids error"),
			},
			wantErr: "injected all group ids error",
		},
		{
			name: "partial_sync_failure",
			sourceGroupClient: &testReadWriteGroupClient{
				groups:       sourceGroups,
				groupMembers: sourceGroupMembers,
				users:        sourceUsers,
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: targetGroups,
				users:  targetUsers,
				groupMembers: map[string][]Member{
					"tg1": {},
					"tg2": {},
				},
			},
			sourceGroupMapper: &testOneToOneGroupMapper{
				m:                sourceGroupMapping,
				mappedGroupIDErr: map[string]error{"sg1": fmt.Errorf("injected mapped group id error")},
			},
			userMapper: &testUserMapper{
				m: userMapping,
			},
			want: map[string][]Member{
				"tg1": {},
				"tg2": {
					&UserMember{Usr: &User{ID: "tu2"}},
				},
			},
			wantErr: "injected mapped group id error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			syncer := NewOneToOneSyncer(&OneToOneSyncerParams{
				Name:              tc.name,
				SourceSystem:      "source",
				TargetSystem:      "target",
				SourceGroupReader: tc.sourceGroupClient,
				TargetGroupWriter: tc.targetGroupClient,
				SourceGroupMapper: tc.sourceGroupMapper,
				UserMapper:        tc.userMapper,
			})

			err := syncer.SyncAll(ctx)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error diff: %v", tc.name, diff)
			}

			for targetGroupID, members := range tc.want {
				got, err := tc.targetGroupClient.GetMembers(ctx, targetGroupID)
				if err != nil {
					t.Fatalf("failed to get target group members for group %s: %v", targetGroupID, err)
				}
				if diff := cmp.Diff(members, got, cmp.AllowUnexported(User{}, Group{})); diff != "" {
					t.Errorf("Process(%+v) got diff (-want, +got): %v", tc.name, diff)
				}
			}
		})
	}
}
