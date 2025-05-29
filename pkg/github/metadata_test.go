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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRoleMetadataStrings(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		metadata      RoleMetadata
		wantStr       string
		wantInviteStr string
	}{
		{
			name:          "member",
			metadata:      RoleMetadata{Role: Member},
			wantStr:       "member",
			wantInviteStr: "direct_member",
		},
		{
			name:          "admin",
			metadata:      RoleMetadata{Role: Admin},
			wantStr:       "admin",
			wantInviteStr: "admin",
		},
		{
			name:          "unspecified",
			metadata:      RoleMetadata{Role: RoleUnspecified},
			wantStr:       "member",
			wantInviteStr: "direct_member",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			roleString := tc.metadata.Role.String()
			inviteString := tc.metadata.Role.InviteString()

			if diff := cmp.Diff(roleString, tc.wantStr); diff != "" {
				t.Errorf("unexpected role string (-got, +want) = %v", diff)
			}
			if diff := cmp.Diff(inviteString, tc.wantInviteStr); diff != "" {
				t.Errorf("unexpected invite string (-got, +want) = %v", diff)
			}
		})
	}
}

func TestRoleMetadataCombine(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		metadataA RoleMetadata
		metadataB RoleMetadata
		want      Role
	}{
		{
			name:      "member+member",
			metadataA: RoleMetadata{Role: Member},
			metadataB: RoleMetadata{Role: Member},
			want:      Member,
		},
		{
			name:      "member+admin",
			metadataA: RoleMetadata{Role: Member},
			metadataB: RoleMetadata{Role: Admin},
			want:      Admin,
		},
		{
			name:      "admin+admin",
			metadataA: RoleMetadata{Role: Admin},
			metadataB: RoleMetadata{Role: Admin},
			want:      Admin,
		},
		{
			name:      "unspecified+member",
			metadataA: RoleMetadata{Role: RoleUnspecified},
			metadataB: RoleMetadata{Role: Member},
			want:      Member,
		},
		{
			name:      "unspecified+admin",
			metadataA: RoleMetadata{Role: RoleUnspecified},
			metadataB: RoleMetadata{Role: Admin},
			want:      Admin,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			combination, ok := tc.metadataA.Combine(&tc.metadataB).(*RoleMetadata)
			if !ok {
				t.Fatalf("could not cast to RoleMetadata")
			}

			if diff := cmp.Diff(combination.Role, tc.want); diff != "" {
				t.Errorf("unexpected role (-got, +want) = %v", diff)
			}
		})
	}
}

func TestNewRoleMetadata(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		roleStr string
		want    *RoleMetadata
	}{
		{
			name:    "member",
			roleStr: "member",
			want:    &RoleMetadata{Role: Member},
		},
		{
			name:    "direct_member",
			roleStr: "direct_member",
			want:    &RoleMetadata{Role: Member},
		},
		{
			name:    "admin",
			roleStr: "admin",
			want:    &RoleMetadata{Role: Admin},
		},
		{
			name:    "unknown",
			roleStr: "hiring_manager",
			want:    &RoleMetadata{Role: Member},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			metadata, ok := NewRoleMetadata(tc.roleStr).(*RoleMetadata)
			if !ok {
				t.Fatalf("could not cast to RoleMetadata")
			}
			if diff := cmp.Diff(metadata, tc.want); diff != "" {
				t.Errorf("unexpected role metadata (-got, +want) = %v", diff)
			}
		})
	}
}
