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

package googlegroups

import (
	"context"
	"fmt"
	"strings"

	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/cloudidentity/v1"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/team-link/pkg/groupsync"
)

const (
	MemberTypeUser  = "USER"
	MemberTypeGroup = "GROUP"
)

// Ensure we conform to the interface.
var _ groupsync.GroupReader = (*GroupReader)(nil)

// GroupReader provides read operations for groups and users in GCP.
type GroupReader struct {
	identity *cloudidentity.Service
	admin    *admin.Service
}

// NewGroupReader create a new GroupReader.
func NewGroupReader(identityService *cloudidentity.Service, adminService *admin.Service) *GroupReader {
	return &GroupReader{
		identity: identityService,
		admin:    adminService,
	}
}

// Descendants retrieve all users (children, recursively) of a group.
func (g GroupReader) Descendants(ctx context.Context, groupID string) ([]*groupsync.User, error) {
	var members []*groupsync.User
	err := g.identity.Groups.Memberships.SearchTransitiveMemberships(groupID).Context(ctx).Pages(ctx,
		func(page *cloudidentity.SearchTransitiveMembershipsResponse) error {
			for _, m := range page.Memberships {
				// we only want user memberships but the API doesn't give us any type information.
				// Instead, we infer the type from the resource name. e.g. groups have a resource
				// name of the form `groups/%s` and users of the form 'users/%d'
				if strings.HasPrefix(m.Member, "users/") {
					members = append(members, &groupsync.User{ID: m.Member})
				}
			}
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch descendants: %w", err)
	}
	return members, nil
}

// GetGroup retrieves the Group with the given ID. The ID must be of the form: groups/{group}.
func (g GroupReader) GetGroup(ctx context.Context, groupID string) (*groupsync.Group, error) {
	group, err := g.identity.Groups.Get(groupID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("could not get group: %w", err)
	}
	return &groupsync.Group{
		ID:         group.Name,
		Attributes: group,
	}, nil
}

// GetMembers retrieves the direct members (children) of the group with given ID.
// This includes both users and subgroups.
func (g GroupReader) GetMembers(ctx context.Context, groupID string) ([]groupsync.Member, error) {
	var members []groupsync.Member
	logger := logging.FromContext(ctx)
	if err := g.identity.Groups.Memberships.List(groupID).Context(ctx).Pages(ctx,
		func(page *cloudidentity.ListMembershipsResponse) error {
			for _, m := range page.Memberships {
				if m.Type == MemberTypeGroup {
					members = append(members, &groupsync.GroupMember{Grp: &groupsync.Group{ID: m.PreferredMemberKey.Id}})
				} else if m.Type == MemberTypeUser {
					members = append(members, &groupsync.UserMember{Usr: &groupsync.User{ID: m.PreferredMemberKey.Id}})
				} else {
					logger.WarnContext(ctx, "unrecognized member type encountered",
						"group_id", groupID,
						"member", m,
					)
				}
			}
			return nil
		},
	); err != nil {
		return nil, fmt.Errorf("could not get group members: %w", err)
	}
	return members, nil
}

// GetUser retrieves the User with the given ID. Should be of the form: users/{userid}.
func (g GroupReader) GetUser(ctx context.Context, userID string) (*groupsync.User, error) {
	user, err := g.admin.Users.Get(userID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("could not get user: %w", err)
	}
	return &groupsync.User{ID: user.Id, Attributes: user}, nil
}
