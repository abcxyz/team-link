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

package github

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v61/github"
	"google.golang.org/protobuf/proto"

	"github.com/abcxyz/pkg/cache"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/sets"
	"github.com/abcxyz/team-link/pkg/groupsync"
)

const (
	IDSep = ":"
	// DefaultCacheDuration is the default time to live for the user and team caches.
	// We don't expect user info (e.g. username etc.) nor team info (team name etc.)
	// to change frequently so a time to live of 1 day is the default.
	DefaultCacheDuration = time.Hour * 24
)

type OrgTokenSource interface {
	// TokenForOrg returns a token that grants access to the given Org's resources.
	TokenForOrg(ctx context.Context, orgID int64) (string, error)
}

type Config struct {
	includeSubTeams         bool
	inviteToOrgIfNotAMember bool
	cacheDuration           time.Duration
}

type Opt func(writer *Config)

// WithoutSubTeamsAsMembers toggles off treating subteams as members of their parent team.
// When this option is used TeamReadWriter.GetMembers will only return user members of the team.
// Similarly, TeamReadWriter.SetMembers will only consider user members when setting members.
func WithoutSubTeamsAsMembers() Opt {
	return func(config *Config) {
		config.includeSubTeams = false
	}
}

// WithCacheDuration set the time to live for the user and team cache entries.
func WithCacheDuration(duration time.Duration) Opt {
	return func(config *Config) {
		config.cacheDuration = duration
	}
}

// WithInviteToOrgIfNotAMember toggles sending an invitation to the user if they are not a
// member of the org being synced to. If the TeamReadWriter is trying to add a user to a team,
// it will first check if they are a member of the org the team belongs. If the user does not
// belong to the org, then the TeamReadWriter will send an invitation to org instead of attempting
// to directly add them to the team.
//
// When enabled, this option may result in several API calls made per user being synced, and thus
// consideration should be made to rate limiting effects when enabling this option.
func WithInviteToOrgIfNotAMember() Opt {
	return func(config *Config) {
		config.inviteToOrgIfNotAMember = true
	}
}

// TeamReadWriter adheres to the groupsync.GroupReadWriter interface
// and provides mechanisms for manipulating GitHub Teams.
type TeamReadWriter struct {
	orgTokenSource          OrgTokenSource
	client                  *github.Client
	userCache               *cache.Cache[*github.User]
	teamCache               *cache.Cache[*github.Team]
	orgMembershipCache      *cache.Cache[bool]
	includeSubTeams         bool
	inviteToOrgIfNotAMember bool
}

// NewTeamReadWriter creates a new TeamReadWriter. By default, TeamReadWriter considers
// subteams as members of their parent team and will treat them as such when executing
// calls to TeamReadWriter.GetMembers and TeamReadWriter.SetMembers. This behavior can
// be disabled by supply the WithoutSubTeamsAsMembers option, in which case only users
// will be considered as members of a team. By default, TeamReadWriter does not attempt
// to add users to an org if they are not already members. This can be enabled by
// WithInviteToOrgIfNotAMember option.
func NewTeamReadWriter(orgTokenSource OrgTokenSource, client *github.Client, opts ...Opt) *TeamReadWriter {
	config := &Config{
		includeSubTeams:         true,
		inviteToOrgIfNotAMember: false,
		cacheDuration:           DefaultCacheDuration,
	}
	for _, opt := range opts {
		opt(config)
	}
	t := &TeamReadWriter{
		orgTokenSource:          orgTokenSource,
		client:                  client,
		includeSubTeams:         config.includeSubTeams,
		inviteToOrgIfNotAMember: config.inviteToOrgIfNotAMember,
		userCache:               cache.New[*github.User](config.cacheDuration),
		teamCache:               cache.New[*github.Team](config.cacheDuration),
		orgMembershipCache:      cache.New[bool](config.cacheDuration),
	}
	return t
}

// GetGroup retrieves the GitHub team with the given ID. The ID must be of the form 'orgID:teamID'.
func (g *TeamReadWriter) GetGroup(ctx context.Context, groupID string) (*groupsync.Group, error) {
	orgID, teamID, err := parseID(groupID)
	if err != nil {
		return nil, fmt.Errorf("could not parse groupID %s: %w", groupID, err)
	}
	client, err := g.githubClientForOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("could not get github client: %w", err)
	}
	team, err := g.getGitHubTeam(ctx, client, orgID, teamID)
	if err != nil {
		return nil, fmt.Errorf("could not get team: %w", err)
	}
	return &groupsync.Group{
		ID:         encode(team.GetOrganization().GetID(), team.GetID()),
		Attributes: team,
	}, nil
}

func (g *TeamReadWriter) getGitHubTeam(ctx context.Context, client *github.Client, orgID, teamID int64) (*github.Team, error) {
	cacheKey := encode(orgID, teamID)
	if team, ok := g.teamCache.Lookup(cacheKey); ok {
		return team, nil
	}
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "fetching team",
		"org_id", orgID,
		"team_id", teamID,
	)
	team, _, err := client.Teams.GetTeamByID(ctx, orgID, teamID)
	if err != nil {
		return nil, fmt.Errorf("could not get team: %w", err)
	}
	g.teamCache.Set(cacheKey, team)
	return team, nil
}

// GetMembers retrieves the direct members (children) of the GitHub team with given ID.
// The ID must be of the form 'orgID:teamID'.
func (g *TeamReadWriter) GetMembers(ctx context.Context, groupID string) ([]groupsync.Member, error) {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "fetching members for team", "team_id", groupID)
	orgID, teamID, err := parseID(groupID)
	if err != nil {
		return nil, fmt.Errorf("could not parse groupID %s: %w", groupID, err)
	}
	client, err := g.githubClientForOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("could not create github client: %w", err)
	}

	users := make(map[string]*github.User, 32)
	if err := paginate(func(listOpts *github.ListOptions) (*github.Response, error) {
		opts := &github.TeamListTeamMembersOptions{
			Role:        "all",
			ListOptions: *listOpts,
		}

		members, resp, err := client.Teams.ListTeamMembersByID(ctx, orgID, teamID, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list team membership: %w", err)
		}

		for _, m := range members {
			// just checking, login should be provided for active members.
			if v := m.GetLogin(); v != "" {
				users[v] = m
			}
		}
		return resp, nil
	}); err != nil {
		return nil, err
	}

	members := make([]groupsync.Member, 0, len(users))
	for _, user := range users {
		members = append(members, &groupsync.UserMember{Usr: &groupsync.User{ID: user.GetLogin(), Attributes: user}})
	}

	if g.includeSubTeams {
		childTeams := make(map[int64]*github.Team, len(users))
		if err := paginate(func(listOpts *github.ListOptions) (*github.Response, error) {
			members, resp, err := client.Teams.ListChildTeamsByParentID(ctx, orgID, teamID, listOpts)
			if err != nil {
				return nil, fmt.Errorf("failed to list team membership: %w", err)
			}

			for _, m := range members {
				if v := m.GetID(); v != 0 {
					childTeams[v] = m
				}
			}
			return resp, nil
		}); err != nil {
			return nil, err
		}
		for _, team := range childTeams {
			members = append(members, &groupsync.GroupMember{Grp: &groupsync.Group{
				ID:         encode(team.GetOrganization().GetID(), team.GetID()),
				Attributes: team,
			}})
		}
	}

	return members, nil
}

// Descendants retrieve all users (children, recursively) of the GitHub team with the given ID.
// The ID must be of the form 'orgID:teamID'.
func (g *TeamReadWriter) Descendants(ctx context.Context, groupID string) ([]*groupsync.User, error) {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "fetching descendants for team", "team_id", groupID)
	users, err := groupsync.Descendants(ctx, groupID, g.GetMembers)
	if err != nil {
		return nil, fmt.Errorf("could not get descendants: %w", err)
	}
	return users, nil
}

// GetUser retrieves the GitHub user with the given ID. The ID is the GitHub user's login.
func (g *TeamReadWriter) GetUser(ctx context.Context, userID string) (*groupsync.User, error) {
	user, err := g.getGitHubUser(ctx, g.client, userID)
	if err != nil {
		return nil, fmt.Errorf("could not get user: %w", err)
	}
	return &groupsync.User{
		ID:         user.GetLogin(),
		Attributes: user,
	}, nil
}

func (g *TeamReadWriter) getGitHubUser(ctx context.Context, client *github.Client, userID string) (*github.User, error) {
	if user, ok := g.userCache.Lookup(userID); ok {
		return user, nil
	}
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "fetching user", "user_id", userID)
	user, _, err := client.Users.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user %s: %w", userID, err)
	}
	g.userCache.Set(userID, user)
	return user, nil
}

// SetMembers replaces the members of the GitHub team with the given ID with the given members.
// The ID must be of the form 'orgID:teamID'. Any members of the GitHub team not found in the given members list
// will be removed. Likewise, any members of the given list that are not currently members of the team will be added.
func (g *TeamReadWriter) SetMembers(ctx context.Context, groupID string, members []groupsync.Member) error {
	orgID, teamID, err := parseID(groupID)
	if err != nil {
		return fmt.Errorf("could not parse groupID %s: %w", groupID, err)
	}
	client, err := g.githubClientForOrg(ctx, orgID)
	if err != nil {
		return fmt.Errorf("could not create github client: %w", err)
	}
	currentMembers, err := g.GetMembers(ctx, groupID)
	if err != nil {
		return fmt.Errorf("could not get current members: %w", err)
	}

	// GitHub usernames and team names are case-insensitive. So we should map each id
	// to lower case before determining who to add and remove.
	currentMemberIDs := toIDMap(currentMembers)
	newMemberIDs := toIDMap(members)

	addMembers := sets.SubtractMapKeys(newMemberIDs, currentMemberIDs)
	removeMembers := sets.SubtractMapKeys(currentMemberIDs, newMemberIDs)

	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "current team members",
		"team_id", groupID,
		"current_member_ids", mapKeys(currentMemberIDs),
	)
	logger.InfoContext(ctx, "authoritative team members",
		"team_id", groupID,
		"authoritative_member_ids", mapKeys(newMemberIDs),
	)
	logger.InfoContext(ctx, "members to add",
		"team_id", groupID,
		"add_member_ids", mapKeys(addMembers),
	)
	logger.InfoContext(ctx, "members to remove",
		"team_id", groupID,
		"remove_member_ids", mapKeys(removeMembers),
	)

	var merr error
	// Add GitHub team memberships.
	for _, member := range addMembers {
		if member.IsUser() {
			user, _ := member.User()
			if err := g.addUserToTeam(ctx, client, orgID, teamID, user.ID); err != nil {
				merr = errors.Join(merr, fmt.Errorf("failed to add user(%s) add user to team(%s): %w", user.ID, groupID, err))
			}
		} else if member.IsGroup() && g.includeSubTeams {
			subteam, _ := member.Group()
			childTeamID, err := validateGroupID(orgID, subteam.ID)
			if err != nil {
				merr = errors.Join(merr, fmt.Errorf("subteamID invalid: %w", err))
				continue
			}
			if err := g.addSubTeamToTeam(ctx, client, orgID, teamID, childTeamID); err != nil {
				merr = errors.Join(merr, fmt.Errorf("failed to add subteam(%s) add user to team(%s): %w", subteam.ID, groupID, err))
			}
		}
	}
	// Remove GitHub team memberships
	for _, member := range removeMembers {
		if member.IsUser() {
			user, _ := member.User()
			if _, err := client.Teams.RemoveTeamMembershipByID(ctx, orgID, teamID, user.ID); err != nil {
				merr = errors.Join(merr, fmt.Errorf("failed to remove user(%s) add user to team(%s): %w", user.ID, groupID, err))
			}
		} else if member.IsGroup() && g.includeSubTeams {
			subteam, _ := member.Group()
			childTeamID, err := validateGroupID(orgID, subteam.ID)
			if err != nil {
				merr = errors.Join(merr, fmt.Errorf("subteamID invalid: %w", err))
				continue
			}
			if err := g.removeSubTeamFromTeam(ctx, client, orgID, teamID, childTeamID); err != nil {
				merr = errors.Join(merr, fmt.Errorf("failed to remove subteam(%s) add user to team(%s): %w", subteam.ID, groupID, err))
			}
		}
	}
	return merr
}

func (g *TeamReadWriter) githubClientForOrg(ctx context.Context, orgID int64) (*github.Client, error) {
	token, err := g.orgTokenSource.TokenForOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get github token: %w", err)
	}
	return g.client.WithAuthToken(token), nil
}

func (g *TeamReadWriter) addUserToTeam(ctx context.Context, client *github.Client, orgID, teamID int64, userID string) error {
	orgIDStr := strconv.FormatInt(orgID, 10)
	isMember, err := g.isOrgMember(ctx, client, orgIDStr, userID)
	if err != nil {
		return fmt.Errorf("could not check if user is a member of organization %d: %w", orgID, err)
	}
	if isMember {
		membershipOpt := &github.TeamAddTeamMembershipOptions{Role: "member"}
		if _, _, err := client.Teams.AddTeamMembershipByID(ctx, orgID, teamID, userID, membershipOpt); err != nil {
			return fmt.Errorf("failed to add GitHub user(%s) for team(%d): %w", userID, teamID, err)
		}
	} else {
		if err := g.inviteToOrg(ctx, client, orgIDStr, teamID, userID); err != nil {
			return fmt.Errorf("failed to invite GitHub user(%s) to org(%d): %w", userID, orgID, err)
		}
	}
	return nil
}

func (g *TeamReadWriter) addSubTeamToTeam(ctx context.Context, client *github.Client, orgID, teamID, childTeamID int64) error {
	if err := addSubTeam(ctx, client, orgID, teamID, childTeamID); err != nil {
		return fmt.Errorf("failed to add child team: %w", err)
	}
	return nil
}

func (g *TeamReadWriter) removeSubTeamFromTeam(ctx context.Context, client *github.Client, orgID, teamID, childTeamID int64) error {
	if err := removeSubTeam(ctx, client, orgID, teamID, childTeamID); err != nil {
		return fmt.Errorf("failed to remove child team: %w", err)
	}
	return nil
}

func (g *TeamReadWriter) isOrgMember(ctx context.Context, client *github.Client, orgID, username string) (bool, error) {
	if !g.inviteToOrgIfNotAMember {
		// if inviting to org is not enabled then we will just assume the user is part of the org
		return true, nil
	}
	cacheKey := fmt.Sprintf("%s:%s", orgID, username)
	if isMember, ok := g.orgMembershipCache.Lookup(cacheKey); ok {
		return isMember, nil
	}
	// check if the user is a member of the org
	isMember, _, err := client.Organizations.IsMember(ctx, orgID, username)
	if err != nil {
		return false, fmt.Errorf("could not check if user is a member of organization %s: %w", orgID, err)
	}
	if isMember {
		// only cache positive memberships as:
		// member -> non-member (ok if this state transition is stale)
		// non-member -> member (we want to recognize this state transition immediately, thus no caching)
		g.orgMembershipCache.Set(cacheKey, isMember)
	}
	return isMember, nil
}

func (g *TeamReadWriter) inviteToOrg(ctx context.Context, client *github.Client, orgID string, teamID int64, username string) error {
	user, err := g.getGitHubUser(ctx, client, username)
	if err != nil {
		return fmt.Errorf("failed to fetch user(%s) info: %w", username, err)
	}
	invitation := &github.CreateOrgInvitationOptions{
		InviteeID: user.ID,
		Role:      proto.String("direct_member"),
		TeamID:    []int64{teamID},
	}
	if _, _, err := client.Organizations.CreateOrgInvitation(ctx, orgID, invitation); err != nil {
		return fmt.Errorf("could not create invitation for user %s to organization %s: %w", username, orgID, err)
	}
	return nil
}

// parseID parses an ID string formatted using encode.
func parseID(groupID string) (int64, int64, error) {
	idComponents := strings.Split(groupID, IDSep)
	if len(idComponents) != 2 {
		return 0, 0, fmt.Errorf("invalid group id: %s", groupID)
	}
	orgID, err := strconv.ParseInt(strings.TrimSpace(idComponents[0]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse %s as a github org ID: %w", idComponents[0], err)
	}
	teamID, err := strconv.ParseInt(strings.TrimSpace(idComponents[1]), 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse %s as a github team ID: %w", idComponents[1], err)
	}
	return orgID, teamID, nil
}

func validateGroupID(orgID int64, groupID string) (int64, error) {
	childOrgID, childTeamID, err := parseID(groupID)
	if err != nil {
		return 0, fmt.Errorf("could not parse group ID %s: %w", groupID, err)
	}
	if childOrgID != orgID {
		return 0, fmt.Errorf("child team orgID must match parent orgID")
	}
	return childTeamID, nil
}

// encode encodes the GitHub org ID and team ID as single ID string.
func encode(orgID, teamID int64) string {
	return fmt.Sprintf("%d%s%d", orgID, IDSep, teamID)
}

func toIDMap(members []groupsync.Member) map[string]groupsync.Member {
	memberIDs := make(map[string]groupsync.Member, len(members))
	for _, m := range members {
		// Convert usernames and team names to lowercase, since they are not
		// case-sensitive in the upstream services:
		//
		//     https://github.com/abcxyz/team-link/pull/63
		id := strings.ToLower(m.ID())
		memberIDs[id] = m
	}
	return memberIDs
}

func addSubTeam(ctx context.Context, client *github.Client, orgID, teamID, subTeamID int64) error {
	subteam, _, err := client.Teams.GetTeamByID(ctx, orgID, subTeamID)
	if err != nil {
		return fmt.Errorf("error fetching team %d: %w", subTeamID, err)
	}
	patch := github.NewTeam{
		Name:         subteam.GetName(),
		ParentTeamID: proto.Int64(teamID),
	}
	_, _, err = client.Teams.EditTeamByID(ctx, orgID, subTeamID, patch, false)
	if err != nil {
		return fmt.Errorf("error adding team %d as a subteam of team %d: %w", subTeamID, teamID, err)
	}
	return nil
}

func removeSubTeam(ctx context.Context, client *github.Client, orgID, teamID, subTeamID int64) error {
	subTeam, _, err := client.Teams.GetTeamByID(ctx, orgID, subTeamID)
	if err != nil {
		return fmt.Errorf("error fetching team %d: %w", subTeamID, err)
	}
	patch := github.NewTeam{
		Name: subTeam.GetName(),
	}
	if _, _, err := client.Teams.EditTeamByID(ctx, orgID, subTeamID, patch, true); err != nil {
		return fmt.Errorf("error removing team %d as a subteam of team %d: %w", subTeamID, teamID, err)
	}
	return nil
}

func mapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
