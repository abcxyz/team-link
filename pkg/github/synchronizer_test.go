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
	"testing"
	"time"

	"github.com/google/go-github/v55/github"
	"github.com/sethvargo/go-retry"

	"github.com/abcxyz/pkg/githubauth"
	"github.com/abcxyz/pkg/sets"
	"github.com/abcxyz/pkg/testutil"
	"github.com/abcxyz/team-link/apis/v1alpha1"
)

const (
	testBadUserName = "bad_user_name"
	testOrgID       = 1
	testTeamID      = 2
)

var (
	testMuxPatternPrefix = fmt.Sprintf("/organizations/%d/team/%d/", testOrgID, testTeamID)
	testUsernames        = []string{"a", "b", "c", "d", testBadUserName}
)

func TestSynchronizer_Sync(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                string
		teamUsernames       []string
		currTeamUsernames   []string
		tokenServerResqCode int
		ListMemberFail      bool
		wantTeamUsernames   []string
		wantSyncErrSubStr   string
	}{
		{
			name:                "success_add_and_remove_members",
			teamUsernames:       []string{"a", "b", "c"},
			currTeamUsernames:   []string{"a", "d"},
			tokenServerResqCode: http.StatusCreated,
			wantTeamUsernames:   []string{"a", "b", "c"},
		},
		{
			name:                "list_member_fail",
			teamUsernames:       []string{"a"},
			tokenServerResqCode: http.StatusCreated,
			ListMemberFail:      true,
			wantSyncErrSubStr:   "failed to get GitHub team members",
		},
		{
			name:                "add_member_fail",
			teamUsernames:       []string{"a", testBadUserName},
			currTeamUsernames:   []string{"a", "d"},
			tokenServerResqCode: http.StatusCreated,
			wantTeamUsernames:   []string{"a"},
			wantSyncErrSubStr:   "failed to add GitHub team members",
		},
		{
			name:                "remove_member_fail",
			teamUsernames:       []string{"a", "b"},
			currTeamUsernames:   []string{"a", testBadUserName},
			tokenServerResqCode: http.StatusCreated,
			wantTeamUsernames:   []string{"a", "b", testBadUserName},
			wantSyncErrSubStr:   "failed to remove GitHub team members",
		},
		{
			name:                "get_access_token_fail",
			teamUsernames:       []string{"a", "b", "c"},
			tokenServerResqCode: http.StatusUnauthorized,
			wantSyncErrSubStr:   "failed to get access token",
		},
	}

	for _, tc := range cases {
		tc := tc
		ctx := context.Background()

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create fake github client.
			ghClient, mux := testSetupGhClient(t)

			// List membership.
			var gotTeamUsernames []string
			bytearr, err := marshal(tc.currTeamUsernames)
			if err != nil {
				t.Fatalf("failed to marshal team usernames: %v", err)
			}
			mux.HandleFunc(fmt.Sprint(testMuxPatternPrefix, "members"), func(w http.ResponseWriter, r *http.Request) {
				if tc.ListMemberFail {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				gotTeamUsernames = append(gotTeamUsernames, tc.currTeamUsernames...)
				fmt.Fprint(w, string(bytearr))
			})

			// Update membership.
			for _, u := range testUsernames {
				mux.HandleFunc(fmt.Sprint(testMuxPatternPrefix, "memberships/", u), func(w http.ResponseWriter, r *http.Request) {
					if u == testBadUserName {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					if r.Method == "PUT" {
						gotTeamUsernames = append(gotTeamUsernames, u)
					} else {
						gotTeamUsernames = remove(gotTeamUsernames, u)
					}
				})
			}

			// Create fake github app.
			ghApp := testNewGitHubApp(t, tc.tokenServerResqCode)

			s := NewSynchronizer(ghClient, ghApp, WithRetry(retry.WithMaxRetries(0, retry.NewFibonacci(500*time.Millisecond))))
			team := convert(tc.teamUsernames)
			gotSyncErr := s.Sync(ctx, team)
			if diff := testutil.DiffErrString(gotSyncErr, tc.wantSyncErrSubStr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error substring: %v", tc.name, diff)
			}
			if !equal(gotTeamUsernames, tc.wantTeamUsernames) {
				t.Errorf("Process(%+v) got unexpected team usernames, got: %v, want: %v", tc.name, gotTeamUsernames, tc.wantTeamUsernames)
			}
		})
	}
}

// testSetupGhClient sets up a test HTTP server along with a github.Client that
// is configured to talk to that test server. Tests should register handlers on
// mux which provide mock responses for the API method being tested.
func testSetupGhClient(tb testing.TB) (*github.Client, *http.ServeMux) {
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

	return client, mux
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

func equal(first, second []string) bool {
	union := sets.Union[string](first, second)
	if len(union) != len(first) || len(union) != len(second) {
		return false
	}
	return true
}

func convert(arr []string) *v1alpha1.GitHubTeam {
	users := make([]*v1alpha1.GitHubUser, len(arr))
	for i, s := range arr {
		users[i] = &v1alpha1.GitHubUser{UserName: s}
	}
	return &v1alpha1.GitHubTeam{
		OrgId:  testOrgID,
		TeamId: testTeamID,
		Users:  users,
	}
}

func marshal(arr []string) ([]byte, error) {
	logins := make([]*github.User, len(arr))
	for i, s := range arr {
		j := i
		logins[j] = &github.User{Login: &s}
	}
	return json.Marshal(logins)
}

func remove(arr []string, item string) []string {
	res := make([]string, 0, len(arr))
	for _, s := range arr {
		if s != item {
			res = append(res, s)
		}
	}
	return res
}
