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
	"github.com/abcxyz/team-link/pkg/groupsync"
)

type Role int

const (
	RoleUnspecified Role = iota
	Member
	Admin
)

const (
	RoleMember = "member"
	RoleAdmin  = "admin"
	// "all" used in queries to get users with all roles.
	RoleAll = "all"
	// github.Invitation has "direct_member" instead of "member" as the role string.
	RoleDirectMember = "direct_member"
)

var roleName = map[Role]string{
	Member:          RoleMember,
	Admin:           RoleAdmin,
	RoleUnspecified: RoleMember,
}

// String gives the string for the role used by GitHub APIs.
func (r Role) String() string {
	return roleName[r]
}

var inviteRoleName = map[Role]string{
	Member:          RoleDirectMember,
	Admin:           RoleAdmin,
	RoleUnspecified: RoleDirectMember,
}

// InviteString gives the string for the role used by the GitHub APIs for Invitations.
// The only difference is that "direct_member" is used instead of "member" for Invitations.
func (r Role) InviteString() string {
	return inviteRoleName[r]
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
		return m
	}
	if m == nil {
		return otherMetadata
	}

	// Take maximum access role granted to the user
	var role Role
	if m.Role > otherMetadata.Role {
		role = m.Role
	} else {
		role = otherMetadata.Role
	}
	return &RoleMetadata{
		Role: role,
	}
}

var stringToRole = map[string]Role{
	RoleMember:       Member,
	RoleDirectMember: Member,
	RoleAdmin:        Admin,
}

func NewRoleMetadata(roleStr string) groupsync.MappingMetadata {
	role, ok := stringToRole[roleStr]
	if !ok {
		// Default to member role if role string is something we don't handle, like "hiring_manager"
		return &RoleMetadata{Role: Member}
	}
	return &RoleMetadata{
		Role: role,
	}
}
