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

	"github.com/abcxyz/team-link/v2/pkg/groupsync"
)

type Role string

const (
	Member          Role = "member"
	Admin           Role = "admin"
	RoleUnspecified Role = Member
)

// List of roles ordered by lowest to highest privileges. Ensure any new roles added are properly ordered.
var Roles []Role = []Role{Member, Admin}

const (
	// "all" used in queries to get users with all roles.
	RoleAll = "all"
	// github.Invitation has "direct_member" instead of "member" as the role string.
	RoleDirectMember = "direct_member"
)

// String gives the string for the role used by GitHub APIs.
func (r Role) String() string {
	return string(r)
}

// InviteString gives the string for the role used by the GitHub APIs for Invitations.
// The only difference is that "direct_member" is used instead of "member" for Invitations.
func (r Role) InviteString() string {
	if r == Member {
		return RoleDirectMember
	}
	return string(r)
}

// RoleMetadata holds a role for a github user being added to
// a target org.
type RoleMetadata struct {
	Role Role
}

// Combine calculates the role for a github user being added to
// a target org by taking the maximum role granted to the user
// via a mapping from a source group.
func (m *RoleMetadata) Combine(other groupsync.MappingMetadata) groupsync.MappingMetadata {
	if other == nil {
		return m
	}
	otherMetadata, ok := other.(*RoleMetadata)
	if !ok {
		// Conversion will fail if other RoleMetadata was not specified in its Mapping object.
		// In this case, just return self metadata.
		return m
	}
	if m == nil {
		return otherMetadata
	}

	// Take maximum access role granted to the user
	role := m.Role
	if omLevel := slices.Index(Roles, otherMetadata.Role); omLevel > slices.Index(Roles, role) {
		role = otherMetadata.Role
	}
	return &RoleMetadata{
		Role: role,
	}
}

func NewRoleMetadata(roleStr string) groupsync.MappingMetadata {
	if slices.Contains(Roles, Role(roleStr)) {
		return &RoleMetadata{Role: Role(roleStr)}
	}
	// Default to member role if role string is something we don't handle, like "hiring_manager"
	return &RoleMetadata{Role: Member}
}
