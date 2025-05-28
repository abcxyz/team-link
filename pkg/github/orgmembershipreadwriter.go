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
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/go-github/v67/github"
	"google.golang.org/protobuf/proto"

	"github.com/abcxyz/pkg/cache"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/sets"
	"github.com/abcxyz/team-link/pkg/groupsync"
	"github.com/abcxyz/team-link/pkg/utils"
)

// OrgMembershipReadWriter adheres to the groupsync.GroupReadWriter interface
// and provides mechanisms for manipulating GitHub org memberships.
type OrgMembershipReadWriter struct {
	orgTokenSource OrgTokenSource
	client         *github.Client
	userCache      *cache.Cache[*github.User]
	orgCache       *cache.Cache[*github.Organization]
	useInvitations bool
}

type OrgRWConfig struct {
	cacheDuration  time.Duration
	useInvitations bool
}

// OrgRWOpt is a configuration option for OrgMembershipReadWriter.
type OrgRWOpt func(writer *OrgRWConfig)

// WithOrgCacheDuration sets the time to live for the user and org cache entries.
func WithOrgCacheDuration(duration time.Duration) OrgRWOpt {
	return func(config *OrgRWConfig) {
		config.cacheDuration = duration
	}
}

// WithoutInvitations avoids using the APIs for Invitations in GitHub.
// GHES does not have invitations APIs so this option is required for GHES.
func WithoutInvitations() OrgRWOpt {
	return func(config *OrgRWConfig) {
		config.useInvitations = false
	}
}

// NewOrgMembershipReadWriter creates a new OrgMembershipReadWriter.
func NewOrgMembershipReadWriter(orgTokenSource OrgTokenSource, client *github.Client, opts ...OrgRWOpt) *OrgMembershipReadWriter {
	config := &OrgRWConfig{
		cacheDuration:  DefaultCacheDuration,
		useInvitations: true,
	}
	for _, opt := range opts {
		opt(config)
	}
	t := &OrgMembershipReadWriter{
		orgTokenSource: orgTokenSource,
		client:         client,
		userCache:      cache.New[*github.User](config.cacheDuration),
		orgCache:       cache.New[*github.Organization](config.cacheDuration),
		useInvitations: config.useInvitations,
	}
	return t
}

// GetGroup retrieves the GitHub org with the given ID.
func (rw *OrgMembershipReadWriter) GetGroup(ctx context.Context, groupID string) (*groupsync.Group, error) {
	orgID, err := strconv.ParseInt(groupID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse orgID from groupID: %s", groupID)
	}
	client, err := rw.githubClientForOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("could not get github client: %w", err)
	}
	org, err := rw.getGitHubOrg(ctx, client, orgID)
	if err != nil {
		return nil, fmt.Errorf("could not get org: %w", err)
	}
	return &groupsync.Group{
		ID:         strconv.FormatInt(org.GetID(), 10),
		Attributes: org,
	}, nil
}

func (rw *OrgMembershipReadWriter) getGitHubOrg(ctx context.Context, client *github.Client, orgID int64) (*github.Organization, error) {
	cacheKey := strconv.FormatInt(orgID, 10)
	if org, ok := rw.orgCache.Lookup(cacheKey); ok {
		return org, nil
	}
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "fetching org",
		"org_id", orgID,
	)
	org, _, err := client.Organizations.GetByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("could not get org: %w", err)
	}
	rw.orgCache.Set(cacheKey, org)
	return org, nil
}

// GetMembers retrieves the members of the GitHub org with given ID.
func (rw *OrgMembershipReadWriter) GetMembers(ctx context.Context, groupID string) ([]groupsync.Member, error) {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "fetching members for org", "org_id", groupID)
	orgID, err := strconv.ParseInt(groupID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse orgID from groupID: %s", groupID)
	}
	client, err := rw.githubClientForOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("could not create github client: %w", err)
	}

	users, err := rw.getUsers(ctx, client, groupID)
	if err != nil {
		return nil, err
	}

	members := make([]groupsync.Member, 0, len(users))
	for _, user := range users {
		members = append(members, &groupsync.UserMember{Usr: &groupsync.User{ID: user.GetLogin(), Attributes: user, Metadata: NewRoleMetadata(user.GetRoleName())}})
	}

	if rw.useInvitations {
		pendingInvites, err := rw.getPendingInvites(ctx, client, groupID)
		if err != nil {
			return nil, err
		}
		for id, invite := range pendingInvites {
			members = append(members, &groupsync.UserMember{Usr: &groupsync.User{ID: id, Attributes: invite, Metadata: NewRoleMetadata(invite.GetRole())}})
		}
	}

	return members, nil
}

// getUsers gets all users in a GitHub org, annotated with RoleName.
func (rw *OrgMembershipReadWriter) getUsers(ctx context.Context, client *github.Client, orgID string) (map[string]*github.User, error) {
	roles := []string{RoleMember, RoleAdmin}
	users := make(map[string]*github.User)
	for _, role := range roles {
		roleUsers, err := rw.getUsersForRole(ctx, client, orgID, role)
		if err != nil {
			return nil, err
		}
		for id, u := range roleUsers {
			users[id] = u
		}
	}
	return users, nil
}

func (rw *OrgMembershipReadWriter) getUsersForRole(ctx context.Context, client *github.Client, orgID, role string) (map[string]*github.User, error) {
	users := make(map[string]*github.User, 32)
	if err := paginate(func(listOpts *github.ListOptions) (*github.Response, error) {
		opts := &github.ListMembersOptions{
			Role:        role,
			ListOptions: *listOpts,
		}

		members, resp, err := client.Organizations.ListMembers(ctx, orgID, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list org membership: %w", err)
		}

		for _, m := range members {
			// Set RoleName ourselves because the ListMembers API does not
			if role != RoleAll {
				m.RoleName = proto.String(role)
			}
			// just checking, login should be provided for active members.
			if v := m.GetLogin(); v != "" {
				users[v] = m
			}
		}
		return resp, nil
	}); err != nil {
		return nil, err
	}
	return users, nil
}

func (rw *OrgMembershipReadWriter) getPendingInvites(ctx context.Context, client *github.Client, orgID string) (map[string]*github.Invitation, error) {
	pendingInvites := make(map[string]*github.Invitation)
	if err := paginate(func(listOpts *github.ListOptions) (*github.Response, error) {
		invites, resp, err := client.Organizations.ListPendingOrgInvitations(ctx, orgID, listOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list pending org invites: %w", err)
		}
		for _, i := range invites {
			// Invites can either have a login (for existing GH users) or an email address (for users who haven't joined GH yet)
			if i.GetLogin() != "" {
				pendingInvites[i.GetLogin()] = i
			} else if i.GetEmail() != "" {
				pendingInvites[i.GetEmail()] = i
			}
		}

		return resp, nil
	}); err != nil {
		return nil, err
	}
	return pendingInvites, nil
}

// Descendants retrieve all users (children, recursively) of the GitHub org with the given ID.
// For orgs, in practice this is the same as GetMembers.
func (rw *OrgMembershipReadWriter) Descendants(ctx context.Context, groupID string) ([]*groupsync.User, error) {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "fetching descendants for org", "org_id", groupID)
	users, err := groupsync.Descendants(ctx, groupID, rw.GetMembers)
	if err != nil {
		return nil, fmt.Errorf("could not get descendants: %w", err)
	}
	return users, nil
}

// GetUser retrieves the GitHub user with the given ID. The ID is the GitHub user's login.
func (rw *OrgMembershipReadWriter) GetUser(ctx context.Context, userID string) (*groupsync.User, error) {
	user, err := rw.getGitHubUser(ctx, rw.client, userID)
	if err != nil {
		return nil, fmt.Errorf("could not get user: %w", err)
	}
	return &groupsync.User{
		ID:         user.GetLogin(),
		Attributes: user,
	}, nil
}

func (rw *OrgMembershipReadWriter) getGitHubUser(ctx context.Context, client *github.Client, userID string) (*github.User, error) {
	if user, ok := rw.userCache.Lookup(userID); ok {
		return user, nil
	}
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "fetching user", "user_id", userID)
	user, _, err := client.Users.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user %s: %w", userID, err)
	}
	rw.userCache.Set(userID, user)
	return user, nil
}

// SetMembers replaces the members of the GitHub org with the given ID with the given members.
// Any members of the GitHub org not found in the given members list will be removed.
// Likewise, any members of the given list that are not currently members of the org will be added.
func (rw *OrgMembershipReadWriter) SetMembers(ctx context.Context, groupID string, members []groupsync.Member) error {
	orgID, err := strconv.ParseInt(groupID, 10, 64)
	if err != nil {
		return fmt.Errorf("could not parse orgID from groupID: %s", groupID)
	}
	client, err := rw.githubClientForOrg(ctx, orgID)
	if err != nil {
		return fmt.Errorf("could not create github client: %w", err)
	}

	currentMembers, err := rw.GetMembers(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to get org members: %w", err)
	}

	// GitHub usernames are case-insensitive. So we should map each id
	// to lower case before determining who to add and remove.
	currentMemberIDs := toIDMap(currentMembers)
	newMemberIDs := toIDMap(members)

	addMembers := sets.SubtractMapKeys(newMemberIDs, currentMemberIDs)
	removeMembers := sets.SubtractMapKeys(currentMemberIDs, newMemberIDs)

	persistentMembers := sets.IntersectMapKeys(newMemberIDs, currentMemberIDs)

	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "current org members",
		"org_id", groupID,
		"current_member_ids", utils.MapKeys(currentMemberIDs),
	)
	logger.InfoContext(ctx, "authoritative org members",
		"org_id", groupID,
		"authoritative_member_ids", utils.MapKeys(newMemberIDs),
	)
	logger.InfoContext(ctx, "members to add",
		"org_id", groupID,
		"add_member_ids", utils.MapKeys(addMembers),
	)
	logger.InfoContext(ctx, "members to remove",
		"org_id", groupID,
		"remove_member_ids", utils.MapKeys(removeMembers),
	)

	var merr error
	// Add GitHub org memberships.
	for _, member := range addMembers {
		if member.IsUser() {
			user, _ := member.User()
			roleMetadata, ok := user.Metadata.(*RoleMetadata)
			if !ok {
				merr = errors.Join(merr, fmt.Errorf("failed to read role metadata for user(%s) in org(%s)", user.ID, groupID))
				continue
			}
			if rw.useInvitations {
				if err := rw.inviteToOrg(ctx, client, groupID, user.ID, roleMetadata.Role); err != nil {
					merr = errors.Join(merr, fmt.Errorf("failed to invite user(%s) to org(%s): %w", user.ID, groupID, err))
				}
			} else {
				if err := rw.setOrgMembershipForUser(ctx, client, groupID, user.ID, roleMetadata.Role); err != nil {
					merr = errors.Join(merr, fmt.Errorf("failed to add user(%s) to org(%s): %w", user.ID, groupID, err))
				}
			}
		}
	}
	// Remove GitHub org memberships
	for _, member := range removeMembers {
		if member.IsUser() {
			user, _ := member.User()
			if _, ok := user.Attributes.(*github.User); ok {
				if _, err := client.Organizations.RemoveOrgMembership(ctx, user.ID, groupID); err != nil {
					merr = errors.Join(merr, fmt.Errorf("failed to remove user(%s) membership from org(%s): %w", user.ID, groupID, err))
				}
			} else if invitation, ok := user.Attributes.(*github.Invitation); rw.useInvitations && ok {
				if invitation.ID == nil {
					merr = errors.Join(merr, fmt.Errorf("failed to extract invitation ID for user(%s) in org(%s)", user.ID, groupID))
					continue
				}
				_, err = client.Organizations.CancelInvite(ctx, groupID, invitation.GetID())
				if err != nil {
					merr = errors.Join(merr, fmt.Errorf("failed to cancel invite(%d) for user(%s) in org(%s): %w", invitation.GetID(), user.ID, groupID, err))
				}
			}
		}
	}
	// Update role for existing users whose role changed
	for id, member := range persistentMembers {
		if member.IsUser() {
			user, _ := member.User()
			newRoleMetdata, hasNewMetadata := user.Metadata.(*RoleMetadata)
			currentUser, _ := currentMemberIDs[id].User()
			currentRoleMetadata, hasCurrentMetadata := currentUser.Metadata.(*RoleMetadata)
			if !hasNewMetadata || !hasCurrentMetadata {
				merr = errors.Join(merr, fmt.Errorf("failed to read role metadata for user(%s) in org(%s)", user.ID, groupID))
				continue
			}
			if newRoleMetdata.Role != currentRoleMetadata.Role {
				if _, isUser := currentUser.Attributes.(*github.User); isUser {
					err := rw.setOrgMembershipForUser(ctx, client, groupID, user.ID, newRoleMetdata.Role)
					if err != nil {
						merr = errors.Join(merr, fmt.Errorf("failed to edit org role for user(%s) in org(%s)", user.ID, groupID))
					}
				} else if invitation, isInvitee := currentUser.Attributes.(*github.Invitation); rw.useInvitations && isInvitee {
					_, err := client.Organizations.CancelInvite(ctx, groupID, invitation.GetID())
					if err != nil {
						merr = errors.Join(merr, fmt.Errorf("failed to cancel invite(%d) in org(%s)", invitation.GetID(), groupID))
					}
					err = rw.inviteToOrg(ctx, client, groupID, user.ID, newRoleMetdata.Role)
					if err != nil {
						merr = errors.Join(merr, fmt.Errorf("failed to add user(%s) to org(%s): %w", user.ID, groupID, err))
					}
				}
			}
		}
	}

	return merr
}

func (rw *OrgMembershipReadWriter) inviteToOrg(ctx context.Context, client *github.Client, orgID, username string, role Role) error {
	user, err := rw.getGitHubUser(ctx, client, username)
	if err != nil {
		return fmt.Errorf("failed to fetch user(%s) info: %w", username, err)
	}
	invitation := &github.CreateOrgInvitationOptions{
		InviteeID: proto.Int64(user.GetID()),
		Role:      proto.String(role.InviteString()),
	}
	if _, _, err := client.Organizations.CreateOrgInvitation(ctx, orgID, invitation); err != nil {
		return fmt.Errorf("could not create invitation for user %s to organization %s: %w", username, orgID, err)
	}
	return nil
}

func (rw *OrgMembershipReadWriter) setOrgMembershipForUser(ctx context.Context, client *github.Client, orgID, user string, role Role) error {
	membership := &github.Membership{
		Role: proto.String(role.String()),
	}
	if _, _, err := client.Organizations.EditOrgMembership(ctx, user, orgID, membership); err != nil {
		return fmt.Errorf("could not set user %s as member in organization %s: %w", user, orgID, err)
	}
	return nil
}

func (rw *OrgMembershipReadWriter) githubClientForOrg(ctx context.Context, orgID int64) (*github.Client, error) {
	token, err := rw.orgTokenSource.TokenForOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get github token: %w", err)
	}
	return rw.client.WithAuthToken(token), nil
}
