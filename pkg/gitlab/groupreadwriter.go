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

package gitlab

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	gitlab "github.com/xanzy/go-gitlab"

	"github.com/abcxyz/pkg/cache"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/sets"
	"github.com/abcxyz/team-link/pkg/groupsync"
	"github.com/abcxyz/team-link/pkg/utils"
)

const (
	// DefaultCacheDuration is the default time to live for the user and group caches.
	// We don't expect user info (e.g. username etc.) nor group info (group name etc.)
	// to change frequently so a time to live of 1 day is the default.
	DefaultCacheDuration = time.Hour * 24
)

type Config struct {
	includeSubGroups bool
	cacheDuration    time.Duration
}

type Opt func(writer *Config)

// WithCacheDuration set the time to live for the user and group cache entries.
func WithCacheDuration(duration time.Duration) Opt {
	return func(config *Config) {
		config.cacheDuration = duration
	}
}

// WithoutSubGroupsAsMembers toggles off treating subgroups as members of their parent group.
// When this option is used GroupReadWriter.GetMembers will only return user members of the group.
// Similarly, GroupReadWriter.SetMembers will only consider user members when setting members.
func WithoutSubGroupsAsMembers() Opt {
	return func(config *Config) {
		config.includeSubGroups = false
	}
}

type GroupReadWriter struct {
	client           *gitlab.Client
	userCache        *cache.Cache[*gitlab.User]
	groupCache       *cache.Cache[*gitlab.Group]
	includeSubGroups bool
}

func NewGroupReadWriter(client *gitlab.Client, opts ...Opt) *GroupReadWriter {
	config := &Config{
		includeSubGroups: true,
		cacheDuration:    DefaultCacheDuration,
	}

	for _, opt := range opts {
		opt(config)
	}
	return &GroupReadWriter{
		client:           client,
		userCache:        cache.New[*gitlab.User](config.cacheDuration),
		groupCache:       cache.New[*gitlab.Group](config.cacheDuration),
		includeSubGroups: config.includeSubGroups,
	}
}

// GetUser retrieves the GitLab user with the given ID. The ID is the GitLab user's login.
func (rw *GroupReadWriter) GetUser(ctx context.Context, userID string) (*groupsync.User, error) {
	user, err := rw.getGitLabUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("could not get user: %w", err)
	}
	return &groupsync.User{
		ID:         user.Username,
		Attributes: user,
	}, nil
}

func (rw *GroupReadWriter) getGitLabUser(ctx context.Context, userID string) (*gitlab.User, error) {
	if user, ok := rw.userCache.Lookup(userID); ok {
		return user, nil
	}
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "fetching user", "user_id", userID)
	users, _, err := rw.client.Users.ListUsers(&gitlab.ListUsersOptions{Username: &userID})
	if err != nil || len(users) == 0 {
		return nil, fmt.Errorf("failed to fetch user %s: %w", userID, err)
	}
	user := users[0]
	rw.userCache.Set(userID, user)
	return user, nil
}

// GetGroup retrieves the GitLab group with the given ID. The ID is the GitLab group's full path, e.g. `foo/bar/baz`.
func (rw *GroupReadWriter) GetGroup(ctx context.Context, groupID string) (*groupsync.Group, error) {
	group, err := rw.getGitLabGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("could not get group: %w", err)
	}
	return &groupsync.Group{
		ID:         strconv.Itoa(group.ID),
		Attributes: group,
	}, nil
}

func (rw *GroupReadWriter) getGitLabGroup(ctx context.Context, groupID string) (*gitlab.Group, error) {
	if group, ok := rw.groupCache.Lookup(groupID); ok {
		return group, nil
	}
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "fetching group", "group_id", groupID)
	id, err := strconv.Atoi(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to convert string group_id to integer: %w", err)
	}
	group, _, err := rw.client.Groups.GetGroup(id, &gitlab.GetGroupOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch group %s: %w", groupID, err)
	}
	rw.groupCache.Set(groupID, group)
	return group, nil
}

func (rw *GroupReadWriter) GetMembers(ctx context.Context, groupID string) ([]groupsync.Member, error) {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "fetching members for group", "group_id", groupID)

	gid, err := strconv.Atoi(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to convert string group_id to integer: %w", err)
	}

	users := make(map[string]*gitlab.GroupMember, 32)
	if err := paginate(func(listOpts *gitlab.ListOptions) (*gitlab.Response, error) {
		userMembers, resp, err := rw.client.Groups.ListGroupMembers(gid, &gitlab.ListGroupMembersOptions{ListOptions: *listOpts})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch group members for %s: %w", groupID, err)
		}

		for _, m := range userMembers {
			users[m.Username] = m
		}
		return resp, nil
	}); err != nil {
		return nil, err
	}

	members := make([]groupsync.Member, 0, len(users))
	for _, user := range users {
		members = append(members, &groupsync.UserMember{Usr: &groupsync.User{ID: user.Username, Attributes: user}})
	}

	if rw.includeSubGroups {
		groups := make(map[string]*gitlab.Group, 32)
		if err := paginate(func(listOpts *gitlab.ListOptions) (*gitlab.Response, error) {
			subgroups, resp, err := rw.client.Groups.ListSubGroups(gid, &gitlab.ListSubGroupsOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to fetch subgroups for %s: %w", groupID, err)
			}

			for _, g := range subgroups {
				groups[strconv.Itoa(g.ID)] = g
			}
			return resp, nil
		}); err != nil {
			return nil, err
		}

		for _, group := range groups {
			members = append(members, &groupsync.GroupMember{Grp: &groupsync.Group{
				ID:         strconv.Itoa(group.ID),
				Attributes: group,
			}})
		}
	}

	return members, nil
}

// Descendants retrieve all users (children, recursively) of the GitLab group with the given ID.
// The ID must be the full path of the group, e.g. `foo/bar/baz`
func (rw *GroupReadWriter) Descendants(ctx context.Context, groupID string) ([]*groupsync.User, error) {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "fetching descendants for group", "group_id", groupID)
	users, err := groupsync.Descendants(ctx, groupID, rw.GetMembers)
	if err != nil {
		return nil, fmt.Errorf("could not get descendants: %w", err)
	}
	return users, nil
}

// SetMembers replaces the members of the GitLab group with the given ID with the given members.
// The ID must be the full path of the group. Any members of the GitLab group not found in the given members list
// will be removed. Likewise, any members of the given list that are not currently members of the group will be added.
func (rw *GroupReadWriter) SetMembers(ctx context.Context, groupID string, members []groupsync.Member) error {
	currentMembers, err := rw.GetMembers(ctx, groupID)
	if err != nil {
		return fmt.Errorf("could not get current members: %w", err)
	}
	currentMemberIDs := toIDMap(currentMembers)
	newMemberIDs := toIDMap(members)

	addMembers := sets.SubtractMapKeys(newMemberIDs, currentMemberIDs)
	removeMembers := sets.SubtractMapKeys(currentMemberIDs, newMemberIDs)

	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "current group members",
		"group_id", groupID,
		"current_member_ids", utils.MapKeys(currentMemberIDs),
	)
	logger.InfoContext(ctx, "authoritative group members",
		"group_id", groupID,
		"authoritative_member_ids", utils.MapKeys(newMemberIDs),
	)
	logger.InfoContext(ctx, "members to add",
		"group_id", groupID,
		"add_member_ids", utils.MapKeys(addMembers),
	)
	logger.InfoContext(ctx, "members to remove",
		"group_id", groupID,
		"remove_member_ids", utils.MapKeys(removeMembers),
	)

	var merr error
	// Add GitLab group memberships.
	for _, member := range addMembers {
		if member.IsUser() {
			user, _ := member.User()
			if err := rw.addUserToGroup(ctx, groupID, user.ID); err != nil {
				merr = errors.Join(merr, err)
			}
		} else if member.IsGroup() && rw.includeSubGroups {
			subgroup, _ := member.Group()
			if err := rw.transferSubGroup(ctx, subgroup, &groupID); err != nil {
				merr = errors.Join(merr, err)
			}
		}
	}
	// Remove GitLab group memberships
	for _, member := range removeMembers {
		if member.IsUser() {
			user, _ := member.User()
			if err := rw.removeUserFromGroup(ctx, groupID, user); err != nil {
				merr = errors.Join(merr, err)
			}
		} else if member.IsGroup() && rw.includeSubGroups {
			subgroup, _ := member.Group()
			// transfer to nil turns the subgroup into a top-level group
			// https://docs.gitlab.com/ee/api/groups.html#transfer-a-group
			if err := rw.transferSubGroup(ctx, subgroup, nil); err != nil {
				merr = errors.Join(merr, err)
			}
		}
	}
	return merr
}

func (rw *GroupReadWriter) addUserToGroup(ctx context.Context, groupID, userID string) error {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "adding user to group", "group_id", groupID, "user_id", userID)

	gid, err := strconv.Atoi(groupID)
	if err != nil {
		return fmt.Errorf("failed to convert string group_id to integer: %w", err)
	}

	_, _, err = rw.client.GroupMembers.AddGroupMember(gid, &gitlab.AddGroupMemberOptions{
		Username:    &userID,
		AccessLevel: utils.ToPtr(gitlab.DeveloperPermissions),
	})
	if err != nil {
		return fmt.Errorf("failed to add GitLab user(%s) for group(%s): %w", userID, groupID, err)
	}
	return nil
}

func (rw *GroupReadWriter) removeUserFromGroup(ctx context.Context, groupID string, user *groupsync.User) error {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "adding user to group", "group_id", groupID, "user_id", user.ID)

	gid, err := strconv.Atoi(groupID)
	if err != nil {
		return fmt.Errorf("failed to convert string group_id to integer: %w", err)
	}

	// extract integer user ID from member attributes because RemoveGroupMember does not support usernames
	memberAttributes, ok := user.Attributes.(*gitlab.GroupMember)
	if !ok {
		return fmt.Errorf("failed to extract GitLab GroupMember attributes from user(%s)", user.ID)
	}
	userID := memberAttributes.ID
	_, err = rw.client.GroupMembers.RemoveGroupMember(gid, userID, &gitlab.RemoveGroupMemberOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove GitLab user(%s) for group(%s): %w", user.ID, groupID, err)
	}
	return nil
}

func (rw *GroupReadWriter) transferSubGroup(ctx context.Context, group *groupsync.Group, newParentGroupID *string) error {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "transferring subgroup to new parent", "group_id", group.ID, "new_parent_id", newParentGroupID)

	groupAttributes, ok := group.Attributes.(*gitlab.Group)
	if !ok {
		return fmt.Errorf("failed to extract GitLab GroupMember attributes from group(%s)", group.ID)
	}
	groupID := groupAttributes.ID
	opts := &gitlab.TransferSubGroupOptions{}
	if newParentGroupID != nil {
		parentGroup, err := rw.getGitLabGroup(ctx, *newParentGroupID)
		if err != nil {
			return fmt.Errorf("failed to get parent group %s: %w", *newParentGroupID, err)
		}
		opts.GroupID = &parentGroup.ID
	}
	_, _, err := rw.client.Groups.TransferSubGroup(groupID, opts)
	if err != nil {
		return fmt.Errorf("failed to transfer GitLab group(%s) to new parent group(%v): %w", group.ID, newParentGroupID, err)
	}
	return nil
}

func toIDMap(members []groupsync.Member) map[string]groupsync.Member {
	memberIDs := make(map[string]groupsync.Member, len(members))
	for _, m := range members {
		memberIDs[m.ID()] = m
	}
	return memberIDs
}
