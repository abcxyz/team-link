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
	"log/slog"
	"strings"

	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/sets"
	"github.com/abcxyz/team-link/pkg/groupsync"
	"github.com/abcxyz/team-link/pkg/utils"
	"github.com/shurcooL/githubv4"
)

type EnterpriseTokenSource interface {
	EnterpriseToken(ctx context.Context) (string, error)
}

var _ groupsync.GroupReadWriter = (*EnterpriseRoleReadWriter)(nil)

type EnterpriseRoleReadWriter struct {
	url         string
	slug        string
	tokenSource EnterpriseTokenSource
	client      *githubv4.Client
	cfg         *entRWConfig
}

type entRWConfig struct {
	permanentMembers map[githubv4.EnterpriseMembershipType]map[string]groupsync.Member
}

// EntRWOpt is an option for configuring the EnterpriseRoleReadWriter.
type EntRWOpt func(*entRWConfig)

// EnterprisePermanentMembers ensures that the GitHub usernames are never
// removed from the specified role.
func EnterprisePermanentMembers(role githubv4.EnterpriseMembershipType, logins []string) EntRWOpt {
	return func(cfg *entRWConfig) {
		members := make(map[string]groupsync.Member, len(logins))
		for _, login := range logins {
			members[login] = &groupsync.UserMember{
				Usr: &groupsync.User{
					ID:       login,
					Metadata: NewRoleMetadata(string(role)),
				},
			}
		}
		cfg.permanentMembers[role] = members
	}
}

// NewEnterpriseRoleReadWriter creates a new EnterpriseRoleReadWriter.
// The url parameter must be in the format of "https://<github_domain>/enterprises/<enterprise_slug>".
func NewEnterpriseRoleReadWriter(url string, tokenSource EnterpriseTokenSource, client *githubv4.Client, opts ...EntRWOpt) (*EnterpriseRoleReadWriter, error) {
	urlSplit := strings.Split(strings.TrimPrefix(url, "https://"), "/")
	if len(urlSplit) != 3 || urlSplit[1] != "enterprises" {
		return nil, fmt.Errorf("invalid enterprise URL: %s", url)
	}

	cfg := &entRWConfig{
		permanentMembers: make(map[githubv4.EnterpriseMembershipType]map[string]groupsync.Member),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return &EnterpriseRoleReadWriter{
		tokenSource: tokenSource,
		client:      client,
		cfg:         cfg,
		url:         url,
		slug:        urlSplit[2],
	}, nil
}

// Descendants returns the users that currently have the enterprise granted.
func (rw *EnterpriseRoleReadWriter) Descendants(ctx context.Context, roleName string) ([]*groupsync.User, error) {
	logger := rw.logger(ctx).With("role_name", roleName)
	logger.InfoContext(ctx, "fetching users with role")

	var query struct {
		Enterprise struct {
			Members struct {
				Nodes []struct {
					Login string
				}
				PageInfo struct {
					EndCursor   githubv4.String
					HasNextPage bool
				}
			} `graphql:"members(role: $role, first: 100, after: $cursor)"`
		} `graphql:"enterprise(slug: $slug)"`
	}
	vars := map[string]any{
		"role":   githubv4.String(roleName),
		"slug":   githubv4.String(rw.slug),
		"cursor": (*githubv4.String)(nil),
	}

	var users []*groupsync.User
	for {
		if err := rw.client.Query(ctx, &query, vars); err != nil {
			return nil, fmt.Errorf("execute GraphQL: %w", err)
		}
		for _, user := range query.Enterprise.Members.Nodes {
			users = append(users, &groupsync.User{
				ID:       user.Login,
				Metadata: NewRoleMetadata(roleName),
			})
		}
		if !query.Enterprise.Members.PageInfo.HasNextPage {
			break
		}
		vars["cursor"] = githubv4.NewString(query.Enterprise.Members.PageInfo.EndCursor)
	}

	logger.InfoContext(ctx, "fetched users", "num_users", len(users))
	return users, nil
}

// GetGroup returns a groupsync.Group for the roleName. No actual lookup is
// performed because GitHub does not support assigning groups to roles at the
// enterprise level, only direct add of users.
func (rw *EnterpriseRoleReadWriter) GetGroup(ctx context.Context, roleName string) (*groupsync.Group, error) {
	return &groupsync.Group{ID: roleName}, nil
}

// GetMembers returns the same list as Descendants, converted to the
// groupsync.Member type.
func (rw *EnterpriseRoleReadWriter) GetMembers(ctx context.Context, roleName string) ([]groupsync.Member, error) {
	users, err := rw.Descendants(ctx, roleName)
	if err != nil {
		return nil, err
	}
	members := make([]groupsync.Member, len(users))
	for i, u := range users {
		members[i] = &groupsync.UserMember{Usr: u}
	}
	return members, nil
}

// GetUser returns the user if it exists in the enterprise.
func (rw *EnterpriseRoleReadWriter) GetUser(ctx context.Context, userID string) (*groupsync.User, error) {
	logger := rw.logger(ctx)
	logger.InfoContext(ctx, "fetching user", "user", userID)

	var query struct {
		User struct {
			Login string
		} `graphql:"user(login: $login)"`
	}
	vars := map[string]any{
		"login": githubv4.String(userID),
	}

	if err := rw.client.Query(ctx, &query, vars); err != nil {
		return nil, fmt.Errorf("execute GraphQL: %w", err)
	}
	return &groupsync.User{ID: query.User.Login}, nil
}

func (rw *EnterpriseRoleReadWriter) SetMembers(ctx context.Context, roleName string, members []groupsync.Member) error {
	role := githubv4.EnterpriseMembershipType(roleName)

	currentMembers, err := rw.GetMembers(ctx, roleName)
	if err != nil {
		return fmt.Errorf("get current members for role %s: %w", roleName, err)
	}

	currentMemberIDs := toIDMap(currentMembers)
	newMemberIDs := toIDMap(members)
	if permanentMembers, ok := rw.cfg.permanentMembers[role]; ok {
		newMemberIDs = sets.UnionMapKeys(newMemberIDs, permanentMembers)
	}

	addMembers := sets.SubtractMapKeys(newMemberIDs, currentMemberIDs)
	removeMembers := sets.SubtractMapKeys(currentMemberIDs, newMemberIDs)

	logger := rw.logger(ctx).With("role_name", role)
	logger.InfoContext(ctx, "current members with the role",
		"current_member_count", len(currentMemberIDs),
		"current_member_ids", utils.MapKeys(currentMemberIDs),
	)
	logger.InfoContext(ctx, "authoritative members who should have the role",
		"authoritative_member_count", len(newMemberIDs),
		"authoritative_member_ids", utils.MapKeys(newMemberIDs),
	)
	logger.InfoContext(ctx, "members to add",
		"add_member_count", len(addMembers),
		"add_member_ids", utils.MapKeys(addMembers),
	)
	logger.InfoContext(ctx, "members to remove",
		"remove_member_count", len(removeMembers),
		"remove_member_ids", utils.MapKeys(removeMembers),
	)

	switch role {
	case githubv4.EnterpriseMembershipTypeAdmin:
		return rw.setAdmins(ctx, addMembers, removeMembers)
	default:
		return fmt.Errorf("unsupported role: %s", role)
	}
}

func (rw *EnterpriseRoleReadWriter) setAdmins(ctx context.Context, add, remove map[string]groupsync.Member) error {
	if !rw.isGHES() {
		return fmt.Errorf("setAdmins: GHC not currently supported")
	}
	return rw.setAdminsGHES(ctx, add, remove)
}

func (rw *EnterpriseRoleReadWriter) setAdminsGHES(ctx context.Context, add, remove map[string]groupsync.Member) error {
	entID, err := rw.fetchEnterpriseID(ctx)
	if err != nil {
		return fmt.Errorf("fetch enterprise ID: %w", err)
	}
	if err := rw.addAdminsGHES(ctx, entID, add); err != nil {
		return fmt.Errorf("add admins: %w", err)
	}
	if err := rw.removeAdminsGHES(ctx, entID, remove); err != nil {
		return fmt.Errorf("remove admins: %w", err)
	}
	return nil
}

func (rw *EnterpriseRoleReadWriter) addAdminsGHES(ctx context.Context, entID githubv4.ID, members map[string]groupsync.Member) error {
	var errs error
	for _, member := range members {
		var m struct {
			AddEnterpriseAdmin struct {
				Admin struct {
					Login string
				}
			} `graphql:"addEnterpriseAdmin(input: $input)"`
		}
		// For some reason, the githubv4 package only has a struct for RemoveEnterpriseAdmin,
		// but the input fields are the same as AddEnterpriseAdmin so it is safe to use here.
		input := githubv4.RemoveEnterpriseAdminInput{
			EnterpriseID: entID,
			Login:        githubv4.String(member.ID()),
		}

		logger := rw.logger(ctx).With("user", member.ID())
		logger.InfoContext(ctx, "granting admin role")
		if err := rw.client.Mutate(ctx, &m, input, nil); err != nil {
			logger.ErrorContext(ctx, "failed to grant admin", "err", err.Error())
			errs = errors.Join(errs, fmt.Errorf("add admin %q: %w", member.ID(), err))
		}
	}
	return errs
}

func (rw *EnterpriseRoleReadWriter) removeAdminsGHES(ctx context.Context, entID githubv4.ID, members map[string]groupsync.Member) error {
	var errs error
	for _, member := range members {
		var m struct {
			RemoveEnterpriseAdmin struct {
				Admin struct {
					Login string
				}
			} `graphql:"removeEnterpriseAdmin(input: $input)"`
		}
		input := githubv4.RemoveEnterpriseAdminInput{
			EnterpriseID: entID,
			Login:        githubv4.String(member.ID()),
		}

		logger := rw.logger(ctx).With("user", member.ID())
		logger.InfoContext(ctx, "removing admin role")
		if err := rw.client.Mutate(ctx, &m, input, nil); err != nil {
			logger.ErrorContext(ctx, "failed to remove admin", "err", err.Error())
			errs = errors.Join(errs, fmt.Errorf("add admin %q: %w", member.ID(), err))
		}
	}
	return errs
}

func (rw *EnterpriseRoleReadWriter) fetchEnterpriseID(ctx context.Context) (githubv4.ID, error) {
	var query struct {
		Enterprise struct {
			ID string
		} `graphql:"enterprise(slug: $slug)"`
	}
	vars := map[string]any{
		"slug": rw.slug,
	}
	if err := rw.client.Query(ctx, &query, vars); err != nil {
		return nil, fmt.Errorf("execute GraphQL: %w", err)
	}
	return githubv4.ID(query.Enterprise.ID), nil
}

func (rw *EnterpriseRoleReadWriter) isGHES() bool {
	return !strings.HasPrefix(rw.url, DefaultGitHubEndpointURL)
}

func (rw *EnterpriseRoleReadWriter) logger(ctx context.Context) *slog.Logger {
	return logging.FromContext(ctx).With("enterprise", rw.url)
}
