package github

import (
	"context"
	"errors"
	"fmt"
	"github.com/abcxyz/pkg/sets"
	"github.com/abcxyz/team-link/pkg/groupsync"
	"github.com/google/go-github/v61/github"
	"strconv"
	"strings"
)

const IDSep = ":"

type OrgTokenSource interface {
	// TokenForOrg returns a token that grants access to the given Org's resources.
	TokenForOrg(ctx context.Context, orgID int64) (string, error)
}

// GroupClient adheres to the groupsync.GroupReadWriter interface
// and provides mechanisms for manipulating GitHub Teams.
type GroupClient struct {
	orgTokenSource OrgTokenSource
	client         *github.Client
}

func NewGroupReadWriter(orgTokenSource OrgTokenSource, client *github.Client) *GroupClient {
	return &GroupClient{
		orgTokenSource: orgTokenSource,
		client:         client,
	}
}

// GetGroup retrieves the GitHub team with the given ID. The ID must be of the form 'orgID:teamID'.
func (g *GroupClient) GetGroup(ctx context.Context, groupID string) (*groupsync.Group, error) {
	orgID, teamID, err := parseID(groupID)
	if err != nil {
		return nil, fmt.Errorf("could not parse groupID %s: %w", groupID, err)
	}
	client, err := g.githubClientForOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("could not get github client: %w", err)
	}
	team, _, err := client.Teams.GetTeamByID(ctx, orgID, teamID)
	if err != nil {
		return nil, fmt.Errorf("could not get team: %w", err)
	}
	group := &groupsync.Group{
		ID:         encode(team.GetOrganization().GetID(), team.GetID()),
		Attributes: team,
	}
	return group, nil
}

// GetMembers retrieves the direct members (children) of the GitHub team with given ID.
// The ID must be of the form 'orgID:teamID'.
func (g *GroupClient) GetMembers(ctx context.Context, groupID string) ([]groupsync.Member, error) {
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
	return members, nil
}

// Descendants retrieve all users (children, recursively) of the GitHub team with the given ID.
// The ID must be of the form 'orgID:teamID'.
func (g *GroupClient) Descendants(ctx context.Context, groupID string) ([]*groupsync.User, error) {
	return groupsync.Descendants(ctx, groupID, g.GetMembers)
}

// GetUser retrieves the GitHub user with the given ID. The ID is the GitHub user's login.
func (g *GroupClient) GetUser(ctx context.Context, userID string) (*groupsync.User, error) {
	ghUser, _, err := g.client.Users.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user %s: %w", userID, err)
	}
	user := &groupsync.User{
		ID:         ghUser.GetLogin(),
		Attributes: ghUser,
	}
	return user, nil
}

// SetMembers replaces the members of the GitHub team with the given ID with the given members.
// The ID must be of the form 'orgID:teamID'. Any members of the GitHub team not found in the given members list
// will be added. Likewise, any members of the given members list that are not currently member of then team will
// be added.
func (g *GroupClient) SetMembers(ctx context.Context, groupID string, members []groupsync.Member) error {
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

	currentLogins := toIDMap(currentMembers)
	memberLogins := toIDMap(members)

	add := sets.SubtractMapKeys(memberLogins, currentLogins)
	remove := sets.SubtractMapKeys(currentLogins, memberLogins)
	var merr error
	// Add GitHub team memberships.
	for _, member := range add {
		if member.IsUser() {
			user, _ := member.User()
			membershipOpt := &github.TeamAddTeamMembershipOptions{Role: "member"}
			_, _, err := client.Teams.AddTeamMembershipByID(ctx, orgID, teamID, user.ID, membershipOpt)
			if err != nil {
				merr = errors.Join(merr, fmt.Errorf("failed to add GitHub team members for team(%d): %w", teamID, err))
			}
		}
	}
	// Remove GitHub team memberships
	for _, member := range remove {
		if member.IsUser() {
			user, _ := member.User()
			// If it is a pending invitation, RemoveTeamMembershipByID will cancel the
			// pending invitation for the team and for that user.
			if _, err := client.Teams.RemoveTeamMembershipByID(ctx, orgID, teamID, user.ID); err != nil {
				merr = errors.Join(merr, fmt.Errorf("failed to remove GitHub team members for team(%d): %w", teamID, err))
			}
		}
	}
	return nil
}

func (g *GroupClient) githubClientForOrg(ctx context.Context, orgID int64) (*github.Client, error) {
	token, err := g.orgTokenSource.TokenForOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get github token: %w", err)
	}
	return g.client.WithAuthToken(token), nil
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

// encode encodes the GitHub org ID and team ID as single ID string.
func encode(orgID int64, teamID int64) string {
	return fmt.Sprintf("%d%s%d", orgID, IDSep, teamID)
}

func toIDMap(members []groupsync.Member) map[string]groupsync.Member {
	memberIDs := make(map[string]groupsync.Member, len(members))
	for _, m := range members {
		if m.IsUser() {
			user, _ := m.User()
			memberIDs[user.ID] = m
		} else {
			group, _ := m.Group()
			memberIDs[group.ID] = m
		}
	}
	return memberIDs
}
