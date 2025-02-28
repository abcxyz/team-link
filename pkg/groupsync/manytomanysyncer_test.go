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
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abcxyz/pkg/testutil"
)

func TestSync(t *testing.T) {
	t.Parallel()

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
			sourceGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"1": {"99", "98"},
					"2": {"97"},
					"3": {"96"},
					"4": {"97"},
					"5": {"98"},
				},
			},
			targetGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"99": {"1"},
					"98": {"1", "5"},
					"97": {"2", "4"},
					"96": {"3"},
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
			sourceGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"1": {"99", "98"},
					"2": {"97"},
					"3": {"96"},
					"4": {"97"},
					"5": {"98"},
				},
			},
			targetGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"99": {"1"},
					"98": {"1", "5"},
					"97": {"2", "4"},
					"96": {"3"},
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
			sourceGroupMapper: &testGroupMapper{},
			targetGroupMapper: &testGroupMapper{},
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
			sourceGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"1": {"99", "98"},
					"2": {"97"},
					"3": {"96"},
					"4": {"97"},
					"5": {"98"},
				},
			},
			targetGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"99": {"1"},
					"97": {"2", "4"},
					"96": {"3"},
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
			wantErr: "error getting associated source group ids: group 98 not mapped",
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
			sourceGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"1": {"99", "98"},
					"2": {"97"},
					"3": {"96"},
					"4": {"97"},
					"5": {"98"},
				},
			},
			targetGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"99": {"1"},
					"98": {"1", "5"},
					"97": {"2", "4"},
					"96": {"3"},
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
			wantErr: "error getting one or more source users",
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
			sourceGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"1": {"99", "98"},
					"2": {"97"},
					"3": {"96"},
					"4": {"97"},
					"5": {"98"},
				},
			},
			targetGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"99": {"1"},
					"98": {"1", "5"},
					"97": {"2", "4"},
					"96": {"3"},
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
				"99": {},
				"98": {},
			},
			wantErr: "error getting one or more source users",
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
			sourceGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"1": {"99", "98"},
					"2": {"97"},
					"3": {"96"},
					"4": {"97"},
					"5": {"98"},
				},
			},
			targetGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"99": {"1"},
					"98": {"1", "5"},
					"97": {"2", "4"},
					"96": {"3"},
				},
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
			wantErr: "error getting one or more target users",
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
			sourceGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"1": {"99", "98"},
					"2": {"97"},
					"3": {"96"},
					"4": {"97"},
					"5": {"98"},
				},
			},
			targetGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"99": {"1"},
					"98": {"1", "5"},
					"97": {"2", "4"},
					"96": {"3"},
				},
			},
			userMapper: &testUserMapper{},
			syncID:     "1",
			want: map[string][]Member{
				"96": {},
				"97": {},
				"98": {},
				"99": {},
			},
			wantErr: "error getting one or more target users",
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
			sourceGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"1": {"99", "98"},
					"2": {"97"},
					"3": {"96"},
					"4": {"97"},
					"5": {"98"},
				},
			},
			targetGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"99": {"1"},
					"98": {"1", "5"},
					"97": {"2", "4"},
					"96": {"3"},
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
			sourceGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"1": {"99", "98"},
					"2": {"97"},
					"3": {"96"},
					"4": {"97"},
					"5": {"98"},
				},
			},
			targetGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"99": {"1"},
					"98": {"1", "5"},
					"97": {"2", "4"},
					"96": {"3"},
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
			sourceGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"1": {"99", "98"},
					"2": {"97"},
					"3": {"96"},
					"4": {"97"},
					"5": {"98"},
				},
			},
			targetGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"99": {"1"},
					"98": {"1", "5"},
					"97": {"2", "4"},
					"96": {"3"},
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
			sourceGroupMapper: &testGroupMapper{
				allGroupIDsErr: fmt.Errorf("allGroupIDsErr"),
			},
			targetGroupMapper: &testGroupMapper{},
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
			sourceGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"1": {"98", "99"},
					"2": {"97"},
					"3": {"96"},
					"4": {"97"},
					"5": {"98"},
				},
				mappedGroupIdsErr: map[string]error{
					"1": fmt.Errorf("mappedGroupIdsErr"),
				},
			},
			targetGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"99": {"1"},
					"98": {"1", "5"},
					"97": {"2", "4"},
					"96": {"3"},
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
			sourceGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"1": {"98", "99"},
					"2": {"97"},
					"3": {"96"},
					"4": {"97"},
					"5": {"98"},
				},
				mappedGroupIdsErr: map[string]error{
					"1": fmt.Errorf("mappedGroupIdsErr"),
					"2": fmt.Errorf("mappedGroupIdsErr"),
					"3": fmt.Errorf("mappedGroupIdsErr"),
					"4": fmt.Errorf("mappedGroupIdsErr"),
					"5": fmt.Errorf("mappedGroupIdsErr"),
				},
			},
			targetGroupMapper: &testGroupMapper{
				m: map[string][]string{
					"99": {"1"},
					"98": {"1", "5"},
					"97": {"2", "4"},
					"96": {"3"},
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

type testReadWriteGroupClient struct {
	groups          map[string]*Group
	groupMembers    map[string][]Member
	users           map[string]*User
	descendantsErrs map[string]error
	getGroupErrs    map[string]error
	getMembersErrs  map[string]error
	getUserErrs     map[string]error
	setMembersErrs  map[string]error
	mutex           sync.RWMutex
}

func (tc *testReadWriteGroupClient) Descendants(ctx context.Context, groupID string) ([]*User, error) {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()
	if err, ok := tc.descendantsErrs[groupID]; ok {
		return nil, err
	}
	return Descendants(ctx, groupID, tc.GetMembers)
}

func (tc *testReadWriteGroupClient) GetGroup(ctx context.Context, groupID string) (*Group, error) {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()
	if err, ok := tc.getGroupErrs[groupID]; ok {
		return nil, err
	}
	group, ok := tc.groups[groupID]
	if !ok {
		return nil, fmt.Errorf("group %s not found", groupID)
	}
	return group, nil
}

func (tc *testReadWriteGroupClient) GetMembers(ctx context.Context, groupID string) ([]Member, error) {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()
	if err, ok := tc.getMembersErrs[groupID]; ok {
		return nil, err
	}
	members, ok := tc.groupMembers[groupID]
	if !ok {
		return nil, fmt.Errorf("group %s not found", groupID)
	}
	return members, nil
}

func (tc *testReadWriteGroupClient) GetUser(ctx context.Context, userID string) (*User, error) {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()
	if err, ok := tc.getUserErrs[userID]; ok {
		return nil, err
	}
	user, ok := tc.users[userID]
	if !ok {
		return nil, fmt.Errorf("user %s not found", userID)
	}
	return user, nil
}

func (tc *testReadWriteGroupClient) SetMembers(ctx context.Context, groupID string, members []Member) error {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	if err, ok := tc.setMembersErrs[groupID]; ok {
		return err
	}
	_, ok := tc.groupMembers[groupID]
	if !ok {
		return fmt.Errorf("group %s not found", groupID)
	}
	// sort members so we have deterministic ordering for comparisons
	sort.Slice(members, func(i, j int) bool {
		u1, err := members[i].User()
		if err != nil {
			// test is broken if this happens
			panic(err)
		}
		u2, err := members[j].User()
		if err != nil {
			// test is broken if this happens
			panic(err)
		}
		return u1.ID < u2.ID
	})
	tc.groupMembers[groupID] = members
	return nil
}

type testGroupMapper struct {
	m                   map[string][]string
	allGroupIDsErr      error
	containsGroupIDErrs map[string]error
	mappedGroupIdsErr   map[string]error
}

func (tgm *testGroupMapper) AllGroupIDs(ctx context.Context) ([]string, error) {
	if tgm.allGroupIDsErr != nil {
		return nil, tgm.allGroupIDsErr
	}
	ids := make([]string, 0, len(tgm.m))
	for id := range tgm.m {
		ids = append(ids, id)
	}
	return ids, nil
}

func (tgm *testGroupMapper) ContainsGroupID(ctx context.Context, groupID string) (bool, error) {
	if err, ok := tgm.containsGroupIDErrs[groupID]; ok {
		return false, err
	}
	_, ok := tgm.m[groupID]
	return ok, nil
}

func (tgm *testGroupMapper) MappedGroupIDs(ctx context.Context, groupID string) ([]string, error) {
	if err, ok := tgm.mappedGroupIdsErr[groupID]; ok {
		return nil, err
	}
	ids, ok := tgm.m[groupID]
	if !ok {
		return nil, fmt.Errorf("group %s not mapped", groupID)
	}
	return ids, nil
}

func (tgm *testGroupMapper) Mappings(ctx context.Context, groupID string) ([]Mapping, error) {
	mappedGroupIDs, err := tgm.MappedGroupIDs(ctx, groupID)
	if err != nil {
		return nil, err
	}
	mappings := make([]Mapping, len(mappedGroupIDs))
	for i, groupID := range mappedGroupIDs {
		mappings[i] = Mapping{
			GroupID: groupID,
		}
	}
	return mappings, nil
}

type testUserMapper struct {
	m                map[string]string
	mappedUserIDErrs map[string]error
}

func (tum *testUserMapper) MappedUserID(ctx context.Context, userID string) (string, error) {
	if err, ok := tum.mappedUserIDErrs[userID]; ok {
		return "", err
	}
	id, ok := tum.m[userID]
	if !ok {
		return "", fmt.Errorf("user %s not mapped", userID)
	}
	return id, nil
}
