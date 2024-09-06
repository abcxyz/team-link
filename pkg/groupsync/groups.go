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
	"errors"
	"fmt"
)

// GroupReader provides read operations for a group system.
type GroupReader interface {
	// Descendants retrieve all users (children, recursively) of a group.
	Descendants(ctx context.Context, groupID string) ([]*User, error)

	// GetGroup retrieves the Group with the given ID.
	GetGroup(ctx context.Context, groupID string) (*Group, error)

	// GetMembers retrieves the direct members (children) of the group with given ID.
	GetMembers(ctx context.Context, groupID string) ([]Member, error)

	// GetUser retrieves the User with the given ID.
	GetUser(ctx context.Context, userID string) (*User, error)
}

// GroupWriter provides write operations for a group system.
type GroupWriter interface {
	// SetMembers replaces the members of the group with the given ID with the given members.
	SetMembers(ctx context.Context, groupID string, members []Member) error
}

// GroupReadWriter provides both read and write operations for a group system.
type GroupReadWriter interface {
	GroupReader
	GroupWriter
}

// OneToManyGroupMapper maps group IDs to lists of group IDs.
type OneToManyGroupMapper interface {
	// AllGroupIDs returns the set of groupIDs being mapped (the key set).
	AllGroupIDs(ctx context.Context) ([]string, error)

	// ContainsGroupID returns whether this mapper contains a mapping for the given group ID.
	ContainsGroupID(ctx context.Context, groupID string) (bool, error)

	// MappedGroupIDs returns the list of group IDs mapped to the given group ID.
	MappedGroupIDs(ctx context.Context, groupID string) ([]string, error)
}

// UserMapper maps a user ID to another user ID.
type UserMapper interface {
	// MappedUserID returns the user ID mapped to the given user ID.
	MappedUserID(ctx context.Context, userID string) (string, error)
}

// User represents a user in a group system.
type User struct {
	// ID is the user's ID in the group system.
	ID string `json:"id,omitempty"`
	// Attributes represent arbitrary attributes about the user
	// in the given group system. This field is typically set by
	// the corresponding GroupReader when retrieving the user.
	Attributes any `json:"attributes,omitempty"`
}

// Group represents a group in a group system.
type Group struct {
	// ID is the group's ID in the group system.
	ID string `json:"id,omitempty"`
	// Attributes represent arbitrary attributes about the group
	// in the given group system. This field is typically set by
	// the corresponding GroupReader when retrieving the group.
	Attributes any `json:"attributes,omitempty"`
}

// Member represents a member of a group. A member may either be
// a User or another Group. An instance of Member will always be
// either a User or a Group but not both.
type Member interface {
	// IsGroup returns whether this Member is a Group.
	IsGroup() bool

	// IsUser returns whether this Member is a User.
	IsUser() bool

	// Group returns the underlying group if this Member is a group.
	// Otherwise, it returns an error.
	Group() (*Group, error)

	// User returns the underlying user if this Member is a user.
	// Otherwise, it returns an error.
	User() (*User, error)
}

// UserMember represents a user membership of a group.
type UserMember struct {
	Usr *User
}

// IsUser returns whether this Member is a User. Always returns true.
func (u *UserMember) IsUser() bool {
	return true
}

// IsGroup returns whether this Member is a Group. Always returns false.
func (u *UserMember) IsGroup() bool {
	return false
}

// Group returns an error.
func (u *UserMember) Group() (*Group, error) {
	return nil, fmt.Errorf("user is not a group")
}

// User returns the underlying user if this Member.
func (u *UserMember) User() (*User, error) {
	return u.Usr, nil
}

// GroupMember represents a group membership of a group.
type GroupMember struct {
	Grp *Group
}

// IsGroup returns whether this Member is a Group. Always returns true.
func (g *GroupMember) IsGroup() bool {
	return true
}

// IsUser returns whether this Member is a User. Always returns false.
func (g *GroupMember) IsUser() bool {
	return false
}

// Group returns the underlying group of this Member.
func (g *GroupMember) Group() (*Group, error) {
	return g.Grp, nil
}

// User returns an error.
func (g *GroupMember) User() (*User, error) {
	return nil, fmt.Errorf("group is not a user")
}

// Descendants retrieve all users (children, recursively) of the given
// group ID using the given memberFunc. This function serves mostly as
// a utility function when implementing ReadGroupClients for when there
// is no special logic for fetching descendants.
func Descendants(ctx context.Context, groupID string, memberFunc func(context.Context, string) ([]Member, error)) ([]*User, error) {
	members, err := memberFunc(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("error fetching group members: %s, %w", groupID, err)
	}
	var users []*User
	var merr error
	for _, member := range members {
		if member.IsUser() {
			user, _ := member.User()
			if user != nil {
				users = append(users, user)
			}
		} else {
			group, _ := member.Group()
			if group != nil {
				flattened, err := Descendants(ctx, group.ID, memberFunc)
				if err != nil {
					merr = errors.Join(merr, err)
					continue
				}
				users = append(users, flattened...)
			}
		}
	}
	return users, merr
}
