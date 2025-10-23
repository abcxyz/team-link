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

package groupsync

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abcxyz/pkg/testutil"
)

func TestManyToManySyncer_Name(t *testing.T) {
	t.Parallel()

	syncer := NewManyToManySyncer(
		"test syncer",
		"source",
		"target",
		&testReadWriteGroupClient{},
		&testReadWriteGroupClient{},
		&testOneToManyGroupMapper{},
		&testOneToManyGroupMapper{},
		&testUserMapper{},
	)

	res := syncer.Name()
	if res != "test syncer" {
		t.Errorf("unexpected name: %q", res)
	}
}

func TestSync(t *testing.T) {
	t.Parallel()

	sourceGroupMapping := map[string][]Mapping{
		"1": {{GroupID: "99"}, {GroupID: "98"}},
		"2": {{GroupID: "97"}},
		"3": {{GroupID: "96"}},
		"4": {{GroupID: "97"}},
		"5": {{GroupID: "98"}},
	}
	targetGroupMapping := map[string][]Mapping{
		"99": {{GroupID: "1"}},
		"98": {{GroupID: "1"}, {GroupID: "5"}},
		"97": {{GroupID: "2"}, {GroupID: "4"}},
		"96": {{GroupID: "3"}},
	}

	cases := []struct {
		name              string
		sourceSystem      string
		targetSystem      string
		sourceGroupClient GroupReader
		targetGroupClient GroupReadWriter
		sourceGroupMapper OneToManyGroupMapper
		targetGroupMapper OneToManyGroupMapper
		userMapper        UserMapper
		syncID            string
		want              map[string][]Member
		wantErr           string
	}{
		{
			name:         "simple_mapping",
			sourceSystem: "source",
			targetSystem: "target",
			sourceGroupClient: &testReadWriteGroupClient{
				groupMembers: map[string][]Member{
					"1": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "b"}},
						&GroupMember{Grp: &Group{ID: "3"}},
					},
					"2": {
						&UserMember{Usr: &User{ID: "c"}},
					},
					"3": {
						&UserMember{Usr: &User{ID: "d"}},
						&UserMember{Usr: &User{ID: "e"}},
					},
					"4": {
						&GroupMember{Grp: &Group{ID: "1"}},
						&GroupMember{Grp: &Group{ID: "2"}},
					},
					"5": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "c"}},
					},
				},
				users: map[string]*User{
					"a": {ID: "a"},
					"b": {ID: "b"},
					"c": {ID: "c"},
					"d": {ID: "d"},
					"e": {ID: "e"},
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"99": {ID: "99"},
					"98": {ID: "98"},
					"97": {ID: "97"},
					"96": {ID: "96"},
				},
				users: map[string]*User{
					"xy": {ID: "xy"},
					"zw": {ID: "zw"},
					"qr": {ID: "qr"},
					"uv": {ID: "uv"},
					"st": {ID: "st"},
				},
				groupMembers: map[string][]Member{
					"99": {},
					"98": {},
					"97": {},
					"96": {},
				},
			},
			sourceGroupMapper: &testOneToManyGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: map[string]string{
					"a": "qr",
					"b": "xy",
					"c": "uv",
					"d": "st",
					"e": "zw",
				},
			},
			syncID: "3",
			want: map[string][]Member{
				"96": {
					&UserMember{Usr: &User{ID: "st"}},
					&UserMember{Usr: &User{ID: "zw"}},
				},
				"97": {},
				"98": {},
				"99": {},
			},
		},
		{
			name:         "many_to_many_mapping",
			sourceSystem: "source",
			targetSystem: "target",
			sourceGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"1": {ID: "1"},
					"2": {ID: "3"},
					"3": {ID: "4"},
					"4": {ID: "5"},
				},
				groupMembers: map[string][]Member{
					"1": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "b"}},
						&GroupMember{Grp: &Group{ID: "3"}},
					},
					"2": {
						&UserMember{Usr: &User{ID: "c"}},
					},
					"3": {
						&UserMember{Usr: &User{ID: "d"}},
						&UserMember{Usr: &User{ID: "e"}},
					},
					"4": {
						&GroupMember{Grp: &Group{ID: "1"}},
						&GroupMember{Grp: &Group{ID: "2"}},
					},
					"5": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "c"}},
					},
				},
				users: map[string]*User{
					"a": {ID: "a"},
					"b": {ID: "b"},
					"c": {ID: "c"},
					"d": {ID: "d"},
					"e": {ID: "e"},
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"99": {ID: "99"},
					"98": {ID: "98"},
					"97": {ID: "97"},
					"96": {ID: "96"},
				},
				users: map[string]*User{
					"xy": {ID: "xy"},
					"zw": {ID: "zw"},
					"qr": {ID: "qr"},
					"uv": {ID: "uv"},
					"st": {ID: "st"},
				},
				groupMembers: map[string][]Member{
					"99": {},
					"98": {},
					"97": {},
					"96": {},
				},
			},
			sourceGroupMapper: &testOneToManyGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: map[string]string{
					"a": "qr",
					"b": "xy",
					"c": "uv",
					"d": "st",
					"e": "zw",
				},
			},
			syncID: "2",
			want: map[string][]Member{
				"96": {},
				// Even though 2 contains one member, "c", we expect 97 to have
				// more than one member. This is because group 4 also maps to 97.
				// Thus, the union of the descendents of 2 and 4 will be mapped
				// to 97.
				"97": {
					&UserMember{Usr: &User{ID: "qr"}},
					&UserMember{Usr: &User{ID: "st"}},
					&UserMember{Usr: &User{ID: "uv"}},
					&UserMember{Usr: &User{ID: "xy"}},
					&UserMember{Usr: &User{ID: "zw"}},
				},
				"98": {},
				"99": {},
			},
		},
		{
			name:              "no_target_ids_found",
			sourceSystem:      "source",
			targetSystem:      "target",
			sourceGroupClient: &testReadWriteGroupClient{},
			targetGroupClient: &testReadWriteGroupClient{},
			sourceGroupMapper: &testOneToManyGroupMapper{},
			targetGroupMapper: &testOneToManyGroupMapper{},
			userMapper:        &testUserMapper{},
			syncID:            "45",
			wantErr:           "error fetching target group IDs",
		},
		{
			name:         "error_getting_associated_source_ids",
			sourceSystem: "source",
			targetSystem: "target",
			sourceGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"1": {ID: "1"},
					"2": {ID: "3"},
					"3": {ID: "4"},
					"4": {ID: "5"},
				},
				groupMembers: map[string][]Member{
					"1": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "b"}},
						&GroupMember{Grp: &Group{ID: "3"}},
					},
					"2": {
						&UserMember{Usr: &User{ID: "c"}},
					},
					"3": {
						&UserMember{Usr: &User{ID: "d"}},
						&UserMember{Usr: &User{ID: "e"}},
					},
					"4": {
						&GroupMember{Grp: &Group{ID: "1"}},
						&GroupMember{Grp: &Group{ID: "2"}},
					},
					"5": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "c"}},
					},
				},
				users: map[string]*User{
					"a": {ID: "a"},
					"b": {ID: "b"},
					"c": {ID: "c"},
					"d": {ID: "d"},
					"e": {ID: "e"},
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"99": {ID: "99"},
					"98": {ID: "98"},
					"97": {ID: "97"},
					"96": {ID: "96"},
				},
				users: map[string]*User{
					"xy": {ID: "xy"},
					"zw": {ID: "zw"},
					"qr": {ID: "qr"},
					"uv": {ID: "uv"},
					"st": {ID: "st"},
				},
				groupMembers: map[string][]Member{
					"99": {},
					"98": {},
					"97": {},
					"96": {},
				},
			},
			sourceGroupMapper: &testOneToManyGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
				mappedGroupIDsErr: map[string]error{
					"98": fmt.Errorf("injected mappedGroupIDsErr"),
				},
			},
			userMapper: &testUserMapper{
				m: map[string]string{
					"a": "qr",
					"b": "xy",
					"c": "uv",
					"d": "st",
					"e": "zw",
				},
			},
			syncID: "1",
			want: map[string][]Member{
				"96": {},
				"97": {},
				"98": {},
				"99": {
					&UserMember{Usr: &User{ID: "qr"}},
					&UserMember{Usr: &User{ID: "st"}},
					&UserMember{Usr: &User{ID: "xy"}},
					&UserMember{Usr: &User{ID: "zw"}},
				},
			},
			wantErr: fmt.Sprintf("error getting associated source groups: %s", "injected mappedGroupIDsErr"),
		},
		{
			name:         "error_getting_source_users_partial",
			sourceSystem: "source",
			targetSystem: "target",
			sourceGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"1": {ID: "1"},
					"2": {ID: "3"},
					"3": {ID: "4"},
					"4": {ID: "5"},
				},
				groupMembers: map[string][]Member{
					"1": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "b"}},
						&GroupMember{Grp: &Group{ID: "3"}},
					},
					"2": {
						&UserMember{Usr: &User{ID: "c"}},
					},
					"3": {
						&UserMember{Usr: &User{ID: "d"}},
						&UserMember{Usr: &User{ID: "e"}},
					},
					"4": {
						&GroupMember{Grp: &Group{ID: "1"}},
						&GroupMember{Grp: &Group{ID: "2"}},
					},
				},
				users: map[string]*User{
					"a": {ID: "a"},
					"b": {ID: "b"},
					"c": {ID: "c"},
					"d": {ID: "d"},
					"e": {ID: "e"},
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"99": {ID: "99"},
					"98": {ID: "98"},
					"97": {ID: "97"},
					"96": {ID: "96"},
				},
				users: map[string]*User{
					"xy": {ID: "xy"},
					"zw": {ID: "zw"},
					"qr": {ID: "qr"},
					"uv": {ID: "uv"},
					"st": {ID: "st"},
				},
				groupMembers: map[string][]Member{
					"99": {},
					"98": {},
					"97": {},
					"96": {},
				},
			},
			sourceGroupMapper: &testOneToManyGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: map[string]string{
					"a": "qr",
					"b": "xy",
					"c": "uv",
					"d": "st",
					"e": "zw",
				},
			},
			syncID: "1",
			want: map[string][]Member{
				"96": {},
				"97": {},
				"98": {},
				"99": {
					&UserMember{Usr: &User{ID: "qr"}},
					&UserMember{Usr: &User{ID: "st"}},
					&UserMember{Usr: &User{ID: "xy"}},
					&UserMember{Usr: &User{ID: "zw"}},
				},
			},
			wantErr: "error getting source users",
		},
		{
			name:         "error_getting_source_users_total",
			sourceSystem: "source",
			targetSystem: "target",
			sourceGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"1": {ID: "1"},
					"2": {ID: "3"},
					"3": {ID: "4"},
					"4": {ID: "5"},
				},
				groupMembers: map[string][]Member{
					"2": {
						&UserMember{Usr: &User{ID: "c"}},
					},
					"3": {
						&UserMember{Usr: &User{ID: "d"}},
						&UserMember{Usr: &User{ID: "e"}},
					},
					"4": {
						&GroupMember{Grp: &Group{ID: "1"}},
						&GroupMember{Grp: &Group{ID: "2"}},
					},
				},
				users: map[string]*User{
					"a": {ID: "a"},
					"b": {ID: "b"},
					"c": {ID: "c"},
					"d": {ID: "d"},
					"e": {ID: "e"},
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"99": {ID: "99"},
					"98": {ID: "98"},
					"97": {ID: "97"},
					"96": {ID: "96"},
				},
				users: map[string]*User{
					"xy": {ID: "xy"},
					"zw": {ID: "zw"},
					"qr": {ID: "qr"},
					"uv": {ID: "uv"},
					"st": {ID: "st"},
				},
				groupMembers: map[string][]Member{
					"99": {},
					"98": {},
					"97": {},
					"96": {},
				},
			},
			sourceGroupMapper: &testOneToManyGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: map[string]string{
					"a": "qr",
					"b": "xy",
					"c": "uv",
					"d": "st",
					"e": "zw",
				},
			},
			syncID: "1",
			want: map[string][]Member{
				"99": {},
				"98": {},
			},
			wantErr: "error getting source users",
		},
		{
			name:         "error_mapping_source_users_partial",
			sourceSystem: "source",
			targetSystem: "target",
			sourceGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"1": {ID: "1"},
					"2": {ID: "3"},
					"3": {ID: "4"},
					"4": {ID: "5"},
				},
				groupMembers: map[string][]Member{
					"1": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "b"}},
						&GroupMember{Grp: &Group{ID: "3"}},
					},
					"2": {
						&UserMember{Usr: &User{ID: "c"}},
					},
					"3": {
						&UserMember{Usr: &User{ID: "d"}},
						&UserMember{Usr: &User{ID: "e"}},
					},
					"4": {
						&GroupMember{Grp: &Group{ID: "1"}},
						&GroupMember{Grp: &Group{ID: "2"}},
					},
					"5": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "c"}},
					},
				},
				users: map[string]*User{
					"a": {ID: "a"},
					"b": {ID: "b"},
					"c": {ID: "c"},
					"d": {ID: "d"},
					"e": {ID: "e"},
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"99": {ID: "99"},
					"98": {ID: "98"},
					"97": {ID: "97"},
					"96": {ID: "96"},
				},
				users: map[string]*User{
					"xy": {ID: "xy"},
					"zw": {ID: "zw"},
					"qr": {ID: "qr"},
					"uv": {ID: "uv"},
					"st": {ID: "st"},
				},
				groupMembers: map[string][]Member{
					"99": {},
					"98": {},
					"97": {},
					"96": {},
				},
			},
			sourceGroupMapper: &testOneToManyGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: map[string]string{
					"d": "st",
					"e": "zw",
				},
			},
			syncID: "1",
			want: map[string][]Member{
				"96": {},
				"97": {},
				"98": {},
				"99": {},
			},
			wantErr: "error getting target users",
		},
		{
			name:         "error_mapping_source_users_total",
			sourceSystem: "source",
			targetSystem: "target",
			sourceGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"1": {ID: "1"},
					"2": {ID: "3"},
					"3": {ID: "4"},
					"4": {ID: "5"},
				},
				groupMembers: map[string][]Member{
					"1": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "b"}},
						&GroupMember{Grp: &Group{ID: "3"}},
					},
					"2": {
						&UserMember{Usr: &User{ID: "c"}},
					},
					"3": {
						&UserMember{Usr: &User{ID: "d"}},
						&UserMember{Usr: &User{ID: "e"}},
					},
					"4": {
						&GroupMember{Grp: &Group{ID: "1"}},
						&GroupMember{Grp: &Group{ID: "2"}},
					},
					"5": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "c"}},
					},
				},
				users: map[string]*User{
					"a": {ID: "a"},
					"b": {ID: "b"},
					"c": {ID: "c"},
					"d": {ID: "d"},
					"e": {ID: "e"},
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"99": {ID: "99"},
					"98": {ID: "98"},
					"97": {ID: "97"},
					"96": {ID: "96"},
				},
				users: map[string]*User{
					"xy": {ID: "xy"},
					"zw": {ID: "zw"},
					"qr": {ID: "qr"},
					"uv": {ID: "uv"},
					"st": {ID: "st"},
				},
				groupMembers: map[string][]Member{
					"99": {},
					"98": {},
					"97": {},
					"96": {},
				},
			},
			sourceGroupMapper: &testOneToManyGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{},
			syncID:     "1",
			want: map[string][]Member{
				"96": {},
				"97": {},
				"98": {},
				"99": {},
			},
			wantErr: "error getting target users",
		},
		{
			name:         "error_setting_members_partial",
			sourceSystem: "source",
			targetSystem: "target",
			sourceGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"1": {ID: "1"},
					"2": {ID: "3"},
					"3": {ID: "4"},
					"4": {ID: "5"},
				},
				groupMembers: map[string][]Member{
					"1": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "b"}},
						&GroupMember{Grp: &Group{ID: "3"}},
					},
					"2": {
						&UserMember{Usr: &User{ID: "c"}},
					},
					"3": {
						&UserMember{Usr: &User{ID: "d"}},
						&UserMember{Usr: &User{ID: "e"}},
					},
					"4": {
						&GroupMember{Grp: &Group{ID: "1"}},
						&GroupMember{Grp: &Group{ID: "2"}},
					},
					"5": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "c"}},
					},
				},
				users: map[string]*User{
					"a": {ID: "a"},
					"b": {ID: "b"},
					"c": {ID: "c"},
					"d": {ID: "d"},
					"e": {ID: "e"},
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"99": {ID: "99"},
					"98": {ID: "98"},
					"97": {ID: "97"},
					"96": {ID: "96"},
				},
				users: map[string]*User{
					"xy": {ID: "xy"},
					"zw": {ID: "zw"},
					"qr": {ID: "qr"},
					"uv": {ID: "uv"},
					"st": {ID: "st"},
				},
				groupMembers: map[string][]Member{
					"99": {},
					"98": {},
					"97": {},
					"96": {},
				},
				setMembersErrs: map[string]error{
					"99": fmt.Errorf("error setting members for group 99"),
				},
			},
			sourceGroupMapper: &testOneToManyGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: map[string]string{
					"a": "qr",
					"b": "xy",
					"c": "uv",
					"d": "st",
					"e": "zw",
				},
			},
			syncID: "1",
			want: map[string][]Member{
				"96": {},
				"97": {},
				"98": {
					&UserMember{Usr: &User{ID: "qr"}},
					&UserMember{Usr: &User{ID: "st"}},
					&UserMember{Usr: &User{ID: "uv"}},
					&UserMember{Usr: &User{ID: "xy"}},
					&UserMember{Usr: &User{ID: "zw"}},
				},
				"99": {},
			},
			wantErr: "error setting members for group 99",
		},
		{
			name:         "error_setting_members_total",
			sourceSystem: "source",
			targetSystem: "target",
			sourceGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"1": {ID: "1"},
					"2": {ID: "3"},
					"3": {ID: "4"},
					"4": {ID: "5"},
				},
				groupMembers: map[string][]Member{
					"1": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "b"}},
						&GroupMember{Grp: &Group{ID: "3"}},
					},
					"2": {
						&UserMember{Usr: &User{ID: "c"}},
					},
					"3": {
						&UserMember{Usr: &User{ID: "d"}},
						&UserMember{Usr: &User{ID: "e"}},
					},
					"4": {
						&GroupMember{Grp: &Group{ID: "1"}},
						&GroupMember{Grp: &Group{ID: "2"}},
					},
					"5": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "c"}},
					},
				},
				users: map[string]*User{
					"a": {ID: "a"},
					"b": {ID: "b"},
					"c": {ID: "c"},
					"d": {ID: "d"},
					"e": {ID: "e"},
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"99": {ID: "99"},
					"98": {ID: "98"},
					"97": {ID: "97"},
					"96": {ID: "96"},
				},
				users: map[string]*User{
					"xy": {ID: "xy"},
					"zw": {ID: "zw"},
					"qr": {ID: "qr"},
					"uv": {ID: "uv"},
					"st": {ID: "st"},
				},
				groupMembers: map[string][]Member{
					"99": {},
					"98": {},
					"97": {},
					"96": {},
				},
				setMembersErrs: map[string]error{
					"98": fmt.Errorf("error setting members for group 98"),
					"99": fmt.Errorf("error setting members for group 99"),
				},
			},
			sourceGroupMapper: &testOneToManyGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: map[string]string{
					"a": "qr",
					"b": "xy",
					"c": "uv",
					"d": "st",
					"e": "zw",
				},
			},
			syncID: "1",
			want: map[string][]Member{
				"96": {},
				"97": {},
				"98": {},
				"99": {},
			},
			wantErr: "error setting members for group",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			syncer := NewManyToManySyncer(
				tc.name,
				tc.sourceSystem,
				tc.targetSystem,
				tc.sourceGroupClient,
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

func TestSyncAll(t *testing.T) {
	t.Parallel()

	sourceGroupMapping := map[string][]Mapping{
		"1": {{GroupID: "99"}, {GroupID: "98"}},
		"2": {{GroupID: "97"}},
		"3": {{GroupID: "96"}},
		"4": {{GroupID: "97"}},
		"5": {{GroupID: "98"}},
	}
	targetGroupMapping := map[string][]Mapping{
		"99": {{GroupID: "1"}},
		"98": {{GroupID: "1"}, {GroupID: "5"}},
		"97": {{GroupID: "2"}, {GroupID: "4"}},
		"96": {{GroupID: "3"}},
	}

	cases := []struct {
		name              string
		sourceSystem      string
		targetSystem      string
		sourceGroupClient GroupReader
		targetGroupClient GroupReadWriter
		sourceGroupMapper OneToManyGroupMapper
		targetGroupMapper OneToManyGroupMapper
		userMapper        UserMapper
		want              map[string][]Member
		wantErr           string
	}{
		{
			name:         "sync_all_success",
			sourceSystem: "source",
			targetSystem: "target",
			sourceGroupClient: &testReadWriteGroupClient{
				groupMembers: map[string][]Member{
					"1": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "b"}},
						&GroupMember{Grp: &Group{ID: "3"}},
					},
					"2": {
						&UserMember{Usr: &User{ID: "c"}},
					},
					"3": {
						&UserMember{Usr: &User{ID: "d"}},
						&UserMember{Usr: &User{ID: "e"}},
					},
					"4": {
						&GroupMember{Grp: &Group{ID: "1"}},
						&GroupMember{Grp: &Group{ID: "2"}},
					},
					"5": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "c"}},
					},
				},
				users: map[string]*User{
					"a": {ID: "a"},
					"b": {ID: "b"},
					"c": {ID: "c"},
					"d": {ID: "d"},
					"e": {ID: "e"},
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"99": {ID: "99"},
					"98": {ID: "98"},
					"97": {ID: "97"},
					"96": {ID: "96"},
				},
				users: map[string]*User{
					"xy": {ID: "xy"},
					"zw": {ID: "zw"},
					"qr": {ID: "qr"},
					"uv": {ID: "uv"},
					"st": {ID: "st"},
				},
				groupMembers: map[string][]Member{
					"99": {},
					"98": {},
					"97": {},
					"96": {},
				},
			},
			sourceGroupMapper: &testOneToManyGroupMapper{
				m: sourceGroupMapping,
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: map[string]string{
					"a": "qr",
					"b": "xy",
					"c": "uv",
					"d": "st",
					"e": "zw",
				},
			},
			want: map[string][]Member{
				"96": {
					&UserMember{Usr: &User{ID: "st"}},
					&UserMember{Usr: &User{ID: "zw"}},
				},
				"97": {
					&UserMember{Usr: &User{ID: "qr"}},
					&UserMember{Usr: &User{ID: "st"}},
					&UserMember{Usr: &User{ID: "uv"}},
					&UserMember{Usr: &User{ID: "xy"}},
					&UserMember{Usr: &User{ID: "zw"}},
				},
				"98": {
					&UserMember{Usr: &User{ID: "qr"}},
					&UserMember{Usr: &User{ID: "st"}},
					&UserMember{Usr: &User{ID: "uv"}},
					&UserMember{Usr: &User{ID: "xy"}},
					&UserMember{Usr: &User{ID: "zw"}},
				},
				"99": {
					&UserMember{Usr: &User{ID: "qr"}},
					&UserMember{Usr: &User{ID: "st"}},
					&UserMember{Usr: &User{ID: "xy"}},
					&UserMember{Usr: &User{ID: "zw"}},
				},
			},
		},
		{
			name:              "no_source_ids_found",
			sourceSystem:      "source",
			targetSystem:      "target",
			sourceGroupClient: &testReadWriteGroupClient{},
			targetGroupClient: &testReadWriteGroupClient{},
			sourceGroupMapper: &testOneToManyGroupMapper{
				allGroupIDsErr: fmt.Errorf("injected allGroupIDsErr"),
			},
			targetGroupMapper: &testOneToManyGroupMapper{},
			userMapper:        &testUserMapper{},
			wantErr:           "error fetching source group IDs",
		},
		{
			name:         "sync_all_partial_failure",
			sourceSystem: "source",
			targetSystem: "target",
			sourceGroupClient: &testReadWriteGroupClient{
				groupMembers: map[string][]Member{
					"1": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "b"}},
						&GroupMember{Grp: &Group{ID: "3"}},
					},
					"2": {
						&UserMember{Usr: &User{ID: "c"}},
					},
					"3": {
						&UserMember{Usr: &User{ID: "d"}},
						&UserMember{Usr: &User{ID: "e"}},
					},
					"4": {
						&GroupMember{Grp: &Group{ID: "1"}},
						&GroupMember{Grp: &Group{ID: "2"}},
					},
					"5": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "c"}},
					},
				},
				users: map[string]*User{
					"a": {ID: "a"},
					"b": {ID: "b"},
					"c": {ID: "c"},
					"d": {ID: "d"},
					"e": {ID: "e"},
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"99": {ID: "99"},
					"98": {ID: "98"},
					"97": {ID: "97"},
					"96": {ID: "96"},
				},
				users: map[string]*User{
					"xy": {ID: "xy"},
					"zw": {ID: "zw"},
					"qr": {ID: "qr"},
					"uv": {ID: "uv"},
					"st": {ID: "st"},
				},
				groupMembers: map[string][]Member{
					"99": {},
					"98": {},
					"97": {},
					"96": {},
				},
			},
			sourceGroupMapper: &testOneToManyGroupMapper{
				m: sourceGroupMapping,
				mappedGroupIDsErr: map[string]error{
					"1": fmt.Errorf("injected mappedGroupIDsErr"),
				},
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: map[string]string{
					"a": "qr",
					"b": "xy",
					"c": "uv",
					"d": "st",
					"e": "zw",
				},
			},
			want: map[string][]Member{
				"96": {
					&UserMember{Usr: &User{ID: "st"}},
					&UserMember{Usr: &User{ID: "zw"}},
				},
				"97": {
					&UserMember{Usr: &User{ID: "qr"}},
					&UserMember{Usr: &User{ID: "st"}},
					&UserMember{Usr: &User{ID: "uv"}},
					&UserMember{Usr: &User{ID: "xy"}},
					&UserMember{Usr: &User{ID: "zw"}},
				},
				"98": {
					&UserMember{Usr: &User{ID: "qr"}},
					&UserMember{Usr: &User{ID: "st"}},
					&UserMember{Usr: &User{ID: "uv"}},
					&UserMember{Usr: &User{ID: "xy"}},
					&UserMember{Usr: &User{ID: "zw"}},
				},
				"99": {},
			},
			wantErr: "failed to sync one or more IDs",
		},
		{
			name:         "sync_all_total_failure",
			sourceSystem: "source",
			targetSystem: "target",
			sourceGroupClient: &testReadWriteGroupClient{
				groupMembers: map[string][]Member{
					"1": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "b"}},
						&GroupMember{Grp: &Group{ID: "3"}},
					},
					"2": {
						&UserMember{Usr: &User{ID: "c"}},
					},
					"3": {
						&UserMember{Usr: &User{ID: "d"}},
						&UserMember{Usr: &User{ID: "e"}},
					},
					"4": {
						&GroupMember{Grp: &Group{ID: "1"}},
						&GroupMember{Grp: &Group{ID: "2"}},
					},
					"5": {
						&UserMember{Usr: &User{ID: "a"}},
						&UserMember{Usr: &User{ID: "c"}},
					},
				},
				users: map[string]*User{
					"a": {ID: "a"},
					"b": {ID: "b"},
					"c": {ID: "c"},
					"d": {ID: "d"},
					"e": {ID: "e"},
				},
			},
			targetGroupClient: &testReadWriteGroupClient{
				groups: map[string]*Group{
					"99": {ID: "99"},
					"98": {ID: "98"},
					"97": {ID: "97"},
					"96": {ID: "96"},
				},
				users: map[string]*User{
					"xy": {ID: "xy"},
					"zw": {ID: "zw"},
					"qr": {ID: "qr"},
					"uv": {ID: "uv"},
					"st": {ID: "st"},
				},
				groupMembers: map[string][]Member{
					"99": {},
					"98": {},
					"97": {},
					"96": {},
				},
			},
			sourceGroupMapper: &testOneToManyGroupMapper{
				m: sourceGroupMapping,
				mappedGroupIDsErr: map[string]error{
					"1": fmt.Errorf("injected mappedGroupIDsErr"),
					"2": fmt.Errorf("injected mappedGroupIDsErr"),
					"3": fmt.Errorf("injected mappedGroupIDsErr"),
					"4": fmt.Errorf("injected mappedGroupIDsErr"),
					"5": fmt.Errorf("injected mappedGroupIDsErr"),
				},
			},
			targetGroupMapper: &testOneToManyGroupMapper{
				m: targetGroupMapping,
			},
			userMapper: &testUserMapper{
				m: map[string]string{
					"a": "qr",
					"b": "xy",
					"c": "uv",
					"d": "st",
					"e": "zw",
				},
			},
			want: map[string][]Member{
				"96": {},
				"97": {},
				"98": {},
				"99": {},
			},
			wantErr: "failed to sync one or more IDs",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			syncer := NewManyToManySyncer(
				tc.name,
				tc.sourceSystem,
				tc.targetSystem,
				tc.sourceGroupClient,
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
