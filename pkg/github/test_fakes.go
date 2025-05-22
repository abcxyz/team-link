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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/google/go-github/v67/github"
	"google.golang.org/protobuf/proto"

	"github.com/abcxyz/team-link/pkg/groupsync"
)

type fakeTokenSource struct {
	orgTokens map[int64]string
}

func (f *fakeTokenSource) TokenForOrg(ctx context.Context, orgID int64) (string, error) {
	return f.orgTokens[orgID], nil
}

type GitHubData struct {
	users       map[string]*github.User
	teams       map[string]map[string]*github.Team
	teamMembers map[string]map[string]map[string]struct{}
	orgs        map[string]*github.Organization
	orgMembers  map[string]map[string]struct{}
}

func githubClient(server *httptest.Server) *github.Client {
	client := github.NewClient(nil)
	baseURL, _ := url.Parse(server.URL + "/")
	client.BaseURL = baseURL
	return client
}

func fakeGitHub(githubData *GitHubData) *httptest.Server {
	mux := http.NewServeMux()
	mux.Handle("GET /users/{username}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := r.PathValue("username")
		user, ok := githubData.users[username]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "user not found")
			return
		}
		jsn, err := json.Marshal(user)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal user")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	mux.Handle("GET /organizations/{org_id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(500)
			fmt.Fprintf(w, "missing or malformed authorization header")
			return
		}
		orgID := r.PathValue("org_id")
		org, ok := githubData.orgs[orgID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "orgID not found")
			return
		}
		jsn, err := json.Marshal(org)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal org")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	mux.Handle("GET /orgs/{org_name}/members", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(500)
			fmt.Fprintf(w, "missing or malformed authorization header")
			return
		}
		orgName := r.PathValue("org_name")
		var org *github.Organization
		for _, o := range githubData.orgs {
			if *o.Name == orgName {
				org = o
				break
			}
		}
		if org == nil {
			w.WriteHeader(404)
			fmt.Fprintf(w, "org %s not found", orgName)
			return
		}
		orgID := strconv.FormatInt(*org.ID, 10)
		members, ok := githubData.orgMembers[orgID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "org %s not found", orgID)
			return
		}
		var users []*github.User
		for username := range members {
			user, ok := githubData.users[username]
			if !ok {
				w.WriteHeader(500)
				fmt.Fprintf(w, "user data inconsistency")
				return
			}
			users = append(users, user)
		}
		jsn, err := json.Marshal(users)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal users")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	mux.Handle("DELETE /orgs/{org_name}/memberships/{username}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(500)
			fmt.Fprintf(w, "missing or malformed authorization header")
			return
		}
		orgName := r.PathValue("org_name")
		var org *github.Organization
		for _, o := range githubData.orgs {
			if *o.Name == orgName {
				org = o
				break
			}
		}
		if org == nil {
			w.WriteHeader(404)
			fmt.Fprintf(w, "org %s not found", orgName)
			return
		}
		orgID := strconv.FormatInt(*org.ID, 10)
		username := r.PathValue("username")
		members, ok := githubData.orgMembers[orgID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "org %s not found", orgID)
			return
		}
		_, ok = githubData.users[username]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "user not found")
			return
		}
		delete(members, username)
		w.WriteHeader(http.StatusNoContent)
	}))
	mux.Handle("POST /orgs/{org_name}/invitations", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(500)
			fmt.Fprintf(w, "missing or malformed authorization header")
			return
		}
		orgName := r.PathValue("org_name")
		var org *github.Organization
		for _, o := range githubData.orgs {
			if *o.Name == orgName {
				org = o
				break
			}
		}
		if org == nil {
			w.WriteHeader(404)
			fmt.Fprintf(w, "org %s not found", orgName)
			return
		}
		orgID := strconv.FormatInt(*org.ID, 10)
		members, ok := githubData.orgMembers[orgID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "org %s not found", orgID)
			return
		}
		payload := github.CreateOrgInvitationOptions{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to read request body")
			return
		}
		userID := payload.InviteeID
		var user *github.User
		for _, u := range githubData.users {
			if *u.ID == *userID {
				user = u
				break
			}
		}
		if user == nil {
			w.WriteHeader(404)
			fmt.Fprintf(w, "user not found from inviteeID: %d", userID)
			return
		}
		members[*user.Login] = struct{}{}
		jsn, err := json.Marshal(github.Invitation{
			ID: proto.Int64(1),
		})
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal response")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	mux.Handle("GET /organizations/{org_id}/team/{team_id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(500)
			fmt.Fprintf(w, "missing or malformed authorization header")
			return
		}
		orgID := r.PathValue("org_id")
		teamID := r.PathValue("team_id")
		teams, ok := githubData.teams[orgID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "orgID not found")
			return
		}
		team, ok := teams[teamID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "team not found")
		}
		jsn, err := json.Marshal(team)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal team")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	mux.Handle("GET /organizations/{org_id}/team/{team_id}/members", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(500)
			fmt.Fprintf(w, "missing or malformed authorization header")
			return
		}
		orgID := r.PathValue("org_id")
		teamID := r.PathValue("team_id")
		teamMembers, ok := githubData.teamMembers[orgID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "orgID not found")
			return
		}
		members, ok := teamMembers[teamID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "team not found")
			return
		}
		var users []*github.User
		for username := range members {
			user, ok := githubData.users[username]
			if !ok {
				w.WriteHeader(500)
				fmt.Fprintf(w, "user data inconsistency")
				return
			}
			users = append(users, user)
		}
		jsn, err := json.Marshal(users)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal users")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	mux.Handle("PUT /organizations/{org_id}/team/{team_id}/memberships/{username}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(500)
			fmt.Fprintf(w, "missing or malformed authorization header")
			return
		}
		orgID := r.PathValue("org_id")
		teamID := r.PathValue("team_id")
		username := strings.ToLower(r.PathValue("username"))
		teamMembers, ok := githubData.teamMembers[orgID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "orgID not found")
			return
		}
		members, ok := teamMembers[teamID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "team not found")
			return
		}
		_, ok = githubData.users[username]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "user not found")
			return
		}
		members[username] = struct{}{}
		respBody := make(map[string]string)
		respBody["url"] = r.URL.String()
		respBody["role"] = "member"
		respBody["state"] = "pending"
		jsn, err := json.Marshal(respBody)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal response")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	mux.Handle("DELETE /organizations/{org_id}/team/{team_id}/memberships/{username}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(500)
			fmt.Fprintf(w, "missing or malformed authorization header")
			return
		}
		orgID := r.PathValue("org_id")
		teamID := r.PathValue("team_id")
		username := strings.ToLower(r.PathValue("username"))
		teamMembers, ok := githubData.teamMembers[orgID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "orgID not found")
			return
		}
		members, ok := teamMembers[teamID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "team not found")
			return
		}
		_, ok = githubData.users[username]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "user not found")
			return
		}
		delete(members, username)
		w.WriteHeader(http.StatusNoContent)
	}))
	mux.Handle("GET /organizations/{org_id}/team/{team_id}/teams", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(500)
			fmt.Fprintf(w, "missing or malformed authorization header")
			return
		}
		orgID := r.PathValue("org_id")
		teamID := r.PathValue("team_id")
		orgTeams, ok := githubData.teams[orgID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "orgID not found")
			return
		}
		var childTeams []*github.Team
		for _, team := range orgTeams {
			if team.Parent != nil && team.Parent.ID != nil {
				parentID := strconv.FormatInt(*team.Parent.ID, 10)
				if parentID == teamID {
					childTeams = append(childTeams, team)
				}
			}
		}
		jsn, err := json.Marshal(childTeams)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal users")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	mux.Handle("PATCH /organizations/{org_id}/team/{team_id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(500)
			fmt.Fprintf(w, "missing or malformed authorization header")
			return
		}
		orgID := r.PathValue("org_id")
		teamID := r.PathValue("team_id")
		payload := make(map[string]any)
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to read request body")
			return
		}
		parentTeamID, ok := payload["parent_team_id"].(float64)
		if ok {
			// this is an add parent operation
			teams, ok := githubData.teams[orgID]
			if !ok {
				w.WriteHeader(404)
				fmt.Fprintf(w, "orgID not found")
				return
			}
			team, ok := teams[teamID]
			if !ok {
				w.WriteHeader(404)
				fmt.Fprintf(w, "team not found")
				return
			}
			parentTeam, ok := teams[strconv.FormatInt(int64(parentTeamID), 10)]
			if !ok {
				w.WriteHeader(404)
				fmt.Fprintf(w, "parent team not found")
				return
			}
			team.Parent = parentTeam
		} else {
			// this is a remove parent operation
			teams, ok := githubData.teams[orgID]
			if !ok {
				w.WriteHeader(404)
				fmt.Fprintf(w, "orgID not found")
				return
			}
			team, ok := teams[teamID]
			if !ok {
				w.WriteHeader(404)
				fmt.Fprintf(w, "team not found")
				return
			}
			team.Parent = nil
		}
		team := githubData.teams[orgID][teamID]
		jsn, err := json.Marshal(team)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "failed to marshal team")
			return
		}
		_, err = w.Write(jsn)
		if err != nil {
			return
		}
	}))
	return httptest.NewServer(mux)
}

func sortByID(members []groupsync.Member) {
	slices.SortFunc(members, func(a, b groupsync.Member) int {
		return strings.Compare(a.ID(), b.ID())
	})
}
