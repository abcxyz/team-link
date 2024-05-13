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
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v61/github"

	"github.com/abcxyz/pkg/githubauth"
	"github.com/abcxyz/pkg/testutil"
	"github.com/abcxyz/team-link/apis/v1alpha1"
)

const (
	testBadUserLogin = "bad_user_name"
	testOrgID        = 1234567
)

var (
	testTeamIDs          = []int64{2345678, 3456789}
	testMuxPatternPrefix = fmt.Sprintf("/organizations/%d/team/", testOrgID)
	testUserLogins       = []string{"test-login-a", "test-login-b", "test-login-c", "test-login-d", testBadUserLogin}
)

type testCase struct {
	name                string
	tokenServerResqCode int
	// For lists below, list[0] is for testTeamIDs[0], list[1] is for
	// testTeamIDs[1].
	teamMemberLoginOrEmails [][]string
	currTeamMemberLogins    [][]string
	currTeamInvitations     [][]string
	listMemberFail          []bool
	listInvitationFail      []bool
	wantTeamMemberLogins    []map[string]struct{}
	wantSyncErrSubStr       string
}

func TestSynchronizer_Sync(t *testing.T) {
	t.Parallel()

	cases := []testCase{
		{
			name: "success_add_and_remove_members",
			teamMemberLoginOrEmails: [][]string{
				{"test-login-a", "test-login-b", "test-login-c"},
				{"test-login-a", "test-login-b"},
			},
			currTeamMemberLogins: [][]string{
				{"test-login-a", "test-login-d"},
				{"test-login-a", "test-login-c"},
			},
			currTeamInvitations: [][]string{
				{"test-login-b"},
				{"test-login-b"},
			},
			tokenServerResqCode: http.StatusCreated,
			wantTeamMemberLogins: []map[string]struct{}{
				{"test-login-a": {}, "test-login-c": {}},
				{"test-login-a": {}},
			},
		},
		{
			name: "success_skip_pending_invitation_to_email",
			teamMemberLoginOrEmails: [][]string{
				{"test-login-a"},
				{},
			},
			currTeamMemberLogins: [][]string{{}, {}},
			currTeamInvitations: [][]string{
				{"test-email@example.com"},
				{},
			},
			tokenServerResqCode: http.StatusCreated,
			wantTeamMemberLogins: []map[string]struct{}{
				{"test-login-a": {}},
				{},
			},
		},
		{
			name: "success_skip_adding_user_email",
			teamMemberLoginOrEmails: [][]string{
				{"test-login-a", "test-email@example.com"},
				{},
			},
			currTeamMemberLogins: [][]string{{}, {}},
			currTeamInvitations:  [][]string{{}, {}},
			tokenServerResqCode:  http.StatusCreated,
			wantTeamMemberLogins: []map[string]struct{}{
				{"test-login-a": {}},
				{},
			},
		},
		{
			name: "list_member_fail",
			teamMemberLoginOrEmails: [][]string{
				{"test-login-a"},
				{"test-login-a"},
			},
			currTeamMemberLogins: [][]string{
				{"test-login-a", "test-login-b"},
				{"test-login-a", "test-login-c"},
			},
			currTeamInvitations: [][]string{
				{},
				{"test-login-b"},
			},
			tokenServerResqCode: http.StatusCreated,
			listMemberFail:      []bool{false, true},
			wantTeamMemberLogins: []map[string]struct{}{
				{"test-login-a": {}},
				nil,
			},
			wantSyncErrSubStr: fmt.Sprintf("failed to get GitHub team members/invitations for team(%d)", testTeamIDs[1]),
		},
		{
			name: "list_invitation_fail",
			teamMemberLoginOrEmails: [][]string{
				{"test-login-a"},
				{"test-login-a"},
			},
			currTeamMemberLogins: [][]string{
				{"test-login-a", "test-login-b"},
				{"test-login-a", "test-login-c"},
			},
			currTeamInvitations: [][]string{
				{},
				{"test-login-b"},
			},
			tokenServerResqCode: http.StatusCreated,
			listInvitationFail:  []bool{false, true},
			wantTeamMemberLogins: []map[string]struct{}{
				{"test-login-a": {}},
				{"test-login-a": {}, "test-login-c": {}},
			},
			wantSyncErrSubStr: fmt.Sprintf("failed to get GitHub team members/invitations for team(%d)", testTeamIDs[1]),
		},
		{
			name: "add_member_fail",
			teamMemberLoginOrEmails: [][]string{
				{"test-login-a"},
				{"test-login-a", "test-login-b", testBadUserLogin},
			},
			currTeamMemberLogins: [][]string{
				{"test-login-a", "test-login-b"},
				{"test-login-a", "test-login-d"},
			},
			currTeamInvitations: [][]string{
				{},
				{"test-login-b"},
			},
			tokenServerResqCode: http.StatusCreated,
			wantTeamMemberLogins: []map[string]struct{}{
				{"test-login-a": {}},
				{"test-login-a": {}},
			},
			wantSyncErrSubStr: fmt.Sprintf("failed to add GitHub team members for team(%d)", testTeamIDs[1]),
		},
		{
			name: "remove_member_fail",
			teamMemberLoginOrEmails: [][]string{
				{"test-login-a"},
				{"test-login-a", "test-login-b", "test-login-c"},
			},
			currTeamMemberLogins: [][]string{
				{"test-login-a", "test-login-b"},
				{"test-login-a", testBadUserLogin},
			},
			currTeamInvitations: [][]string{
				{},
				{"test-login-c"},
			},
			tokenServerResqCode: http.StatusCreated,
			wantTeamMemberLogins: []map[string]struct{}{
				{"test-login-a": {}},
				{"test-login-a": {}, testBadUserLogin: {}, "test-login-b": {}},
			},
			wantSyncErrSubStr: fmt.Sprintf("failed to remove GitHub team members for team(%d)", testTeamIDs[1]),
		},
		{
			name:                    "get_access_token_fail",
			teamMemberLoginOrEmails: [][]string{{"test-login-a", "test-login-b", "test-login-c"}},
			tokenServerResqCode:     http.StatusUnauthorized,
			wantTeamMemberLogins:    []map[string]struct{}{nil},
			wantSyncErrSubStr:       "failed to get access token",
		},
	}

	for _, tc := range cases {
		tc := tc
		ctx := context.Background()

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create fake github client.
			gotTeamMemberLogins := make([]map[string]struct{}, len(tc.teamMemberLoginOrEmails))
			teams := make([]*v1alpha1.GitHubTeam, len(tc.teamMemberLoginOrEmails))
			ghClient := testGitHubClient(t, tc, gotTeamMemberLogins, teams)

			// Create fake github app.
			ghApp := testNewGitHubApp(t, tc.tokenServerResqCode)

			s := NewSynchronizer(ghClient, ghApp)
			gotSyncErr := s.Sync(ctx, teams)
			if diff := testutil.DiffErrString(gotSyncErr, tc.wantSyncErrSubStr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error substring: %v", tc.name, diff)
			}
			if diff := cmp.Diff(tc.wantTeamMemberLogins, gotTeamMemberLogins); diff != "" {
				t.Errorf("Process(%+v) got unexpected team member logins (-want,+got):\n%s", tc.name, diff)
			}
		})
	}
}

// testGitHubClient sets up a test HTTP server along with a github.Client that
// is configured to talk to that test server. It also register handlers on mux
// given a test case which provide mock responses for the API method being
// tested, and fills the gotTeamMemberLogins and teams to be synced.
// TODO(#9): instead of mock, use a fake client instead.
func testGitHubClient(
	tb testing.TB,
	tc testCase,
	gotTeamMemberLogins []map[string]struct{},
	teams []*v1alpha1.GitHubTeam,
) *github.Client {
	tb.Helper()

	// mux is the HTTP request multiplexer used with the test server.
	mux := http.NewServeMux()

	apiHandler := http.NewServeMux()
	apiHandler.Handle("/api-v3/", http.StripPrefix("/api-v3", mux))

	// server is a test HTTP server used to provide mock API responses.
	server := httptest.NewServer(apiHandler)
	tb.Cleanup(func() {
		server.Close()
	})

	// client is the GitHub client being tested and is
	// configured to use test server.
	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/api-v3/")
	client.BaseURL = url
	client.UploadURL = url

	// List active memberships.
	for i, logins := range tc.currTeamMemberLogins {
		currTeamMemberLoginsBytes := testJSONMarshalGitHubUser(tb, logins)
		mux.HandleFunc(
			fmt.Sprint(testMuxPatternPrefix, testTeamIDs[i], "/members"),
			func(w http.ResponseWriter, r *http.Request) {
				if len(tc.listMemberFail) > i && tc.listMemberFail[i] {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				gotTeamMemberLogins[i] = make(map[string]struct{}, len(logins))
				for _, l := range logins {
					gotTeamMemberLogins[i][l] = struct{}{}
				}
				fmt.Fprint(w, string(currTeamMemberLoginsBytes))
			},
		)
	}

	// List pending memberships.
	for i, invites := range tc.currTeamInvitations {
		currTeamInvitationsBytes := testJSONMarshalGitHubUser(tb, invites)
		mux.HandleFunc(
			fmt.Sprint(testMuxPatternPrefix, testTeamIDs[i], "/invitations"),
			func(w http.ResponseWriter, r *http.Request) {
				if len(tc.listInvitationFail) > i && tc.listInvitationFail[i] {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				fmt.Fprint(w, string(currTeamInvitationsBytes))
			},
		)
	}

	// Update membership.
	for i, ls := range tc.teamMemberLoginOrEmails {
		for _, u := range testUserLogins {
			mux.HandleFunc(
				fmt.Sprint(testMuxPatternPrefix, testTeamIDs[i], "/memberships/", u),
				func(w http.ResponseWriter, r *http.Request) {
					if u == testBadUserLogin {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					if r.Method == http.MethodPut {
						gotTeamMemberLogins[i][u] = struct{}{}
					} else {
						delete(gotTeamMemberLogins[i], u)
					}
				},
			)
		}
		teams[i] = githubTeam(ls, testTeamIDs[i])
	}

	return client
}

func testNewGitHubApp(tb testing.TB, statusCode int) *githubauth.App {
	tb.Helper()

	ser := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		fmt.Fprintf(w, `{"token":"this-is-the-token-from-github"}`)
	}))
	tb.Cleanup(func() {
		ser.Close()
	})

	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		tb.Fatalf("failed to generate rsa private key: %v", err)
	}

	ghAppOpts := []githubauth.Option{
		githubauth.WithJWTTokenCaching(1 * time.Minute),
		githubauth.WithAccessTokenURLPattern(ser.URL + "/%s/access_tokens"),
	}
	ghApp, err := githubauth.NewApp("test-github-id", "test-github-id", pk, ghAppOpts...)
	if err != nil {
		tb.Fatalf("failed to create github app: %v", err)
	}
	return ghApp
}

// Create a GitHub team with users given team ID and a list of mixed email and
// login of users.The string will be treated as a GitHub user login unless it
// contains '@' which will be treated as a GitHub user email.
func githubTeam(arr []string, teamID int64) *v1alpha1.GitHubTeam {
	users := make([]*v1alpha1.GitHubUser, len(arr))
	for i, s := range arr {
		if strings.Contains(s, "@") {
			users[i] = &v1alpha1.GitHubUser{Email: s}
		} else {
			users[i] = &v1alpha1.GitHubUser{Login: s}
		}
	}
	return &v1alpha1.GitHubTeam{
		OrgId:  testOrgID,
		TeamId: teamID,
		Users:  users,
	}
}

// Convert a list of mixed email and login of users into a list of GitHub users
// and then marshal them to Json. The string will be treated as a GitHub user
// login unless it contains '@' which will be treated as a GitHub user email.
func testJSONMarshalGitHubUser(tb testing.TB, arr []string) []byte {
	tb.Helper()

	logins := make([]*github.User, len(arr))
	for i, s := range arr {
		if strings.Contains(s, "@") {
			logins[i] = &github.User{Email: &s}
		} else {
			//nolint:exportloopref // loop variable is not reused in https://tip.golang.org/doc/go1.22.
			logins[i] = &github.User{Login: &s} //#nosec G601 // loop variable is not reused.
		}
	}
	res, err := json.Marshal(logins)
	if err != nil {
		tb.Fatalf("failed to marshal team member logins: %v", err)
	}
	return res
}
