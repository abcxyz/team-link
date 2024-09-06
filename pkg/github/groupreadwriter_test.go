package github

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/abcxyz/pkg/testutil"
	"github.com/abcxyz/team-link/pkg/groupsync"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v61/github"
	"google.golang.org/protobuf/proto"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
)

func TestGroupClient_GetGroup(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		tokenSource OrgTokenSource
		data        *GitHubData
		groupID     string
		want        *groupsync.Group
		wantErr     string
	}{
		{
			name: "success",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data: &GitHubData{
				teams: map[string]map[string]*github.Team{
					"8583": { // org1
						"2797": &github.Team{
							ID:   proto.Int64(2797),
							Name: proto.String("team1"),
							Organization: &github.Organization{
								ID:   proto.Int64(8583),
								Name: proto.String("org1"),
							},
						},
						"9350": &github.Team{
							ID:   proto.Int64(9350),
							Name: proto.String("team2"),
							Organization: &github.Organization{
								ID:   proto.Int64(8583),
								Name: proto.String("org1"),
							},
						},
					},
					"4701": { // org2
						"3387": &github.Team{
							ID:   proto.Int64(3387),
							Name: proto.String("team3"),
							Organization: &github.Organization{
								ID:   proto.Int64(4701),
								Name: proto.String("org2"),
							},
						},
					},
				},
			},
			groupID: "8583:2797",
			want: &groupsync.Group{
				ID: "8583:2797",
				Attributes: &github.Team{
					ID:   proto.Int64(2797),
					Name: proto.String("team1"),
					Organization: &github.Organization{
						ID:   proto.Int64(8583),
						Name: proto.String("org1"),
					},
				},
			},
		},
		{
			name: "id_wrong_format",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data:    &GitHubData{},
			groupID: "invalidID",
			wantErr: "could not parse groupID invalidID",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			server := fakeGitHub(tc.data)
			defer server.Close()

			client := githubClient(server)

			groupRW := NewGroupReadWriter(tc.tokenSource, client)

			got, err := groupRW.GetGroup(ctx, tc.groupID)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error : %v", err)
			}

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("unexpected gotMembers (-got, +want) = %v", diff)
			}
		})
	}
}

func TestGroupClient_GetMembers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		tokenSource OrgTokenSource
		data        *GitHubData
		groupID     string
		want        []groupsync.Member
		wantErr     string
	}{
		{
			name: "success",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data: &GitHubData{
				users: map[string]*github.User{
					"user1": {
						ID:    proto.Int64(2286),
						Login: proto.String("user1"),
						Email: proto.String("user1@example.com"),
					},
					"user2": {
						ID:    proto.Int64(5660),
						Login: proto.String("user2"),
						Email: proto.String("user2@example.com"),
					},
					"user3": {
						ID:    proto.Int64(3208),
						Login: proto.String("user3"),
						Email: proto.String("user3@example.com"),
					},
				},
				teams: map[string]map[string]*github.Team{
					"8583": { // org1
						"2797": &github.Team{
							ID:   proto.Int64(2797),
							Name: proto.String("team1"),
							Organization: &github.Organization{
								ID:   proto.Int64(8583),
								Name: proto.String("org1"),
							},
						},
						"9350": &github.Team{
							ID:   proto.Int64(9350),
							Name: proto.String("team2"),
							Organization: &github.Organization{
								ID:   proto.Int64(8583),
								Name: proto.String("org1"),
							},
						},
					},
					"4701": { // org2
						"3387": &github.Team{
							ID:   proto.Int64(3387),
							Name: proto.String("team3"),
							Organization: &github.Organization{
								ID:   proto.Int64(4701),
								Name: proto.String("org2"),
							},
						},
					},
				},
				teamMembers: map[string]map[string]map[string]struct{}{
					"8583": { // org1
						"2797": {
							"user2": struct{}{},
						},
						"9350": {
							"user1": struct{}{},
							"user3": struct{}{},
						},
					},
					"4701": { // org2
						"3387": {
							"user1": struct{}{},
						},
					},
				},
			},
			groupID: "8583:9350",
			want: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:    proto.Int64(2286),
							Login: proto.String("user1"),
							Email: proto.String("user1@example.com"),
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &github.User{
							ID:    proto.Int64(3208),
							Login: proto.String("user3"),
							Email: proto.String("user3@example.com"),
						},
					},
				},
			},
		},
		{
			name: "id_wrong_format",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data:    &GitHubData{},
			groupID: "invalidID",
			wantErr: "could not parse groupID invalidID",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			server := fakeGitHub(tc.data)
			defer server.Close()

			client := githubClient(server)

			groupRW := NewGroupReadWriter(tc.tokenSource, client)

			got, err := groupRW.GetMembers(ctx, tc.groupID)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error : %v", err)
			}

			// sort so we have a consistent ordering for comparison
			sortByID(got)

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("unexpected gotMembers (-got, +want) = %v", diff)
			}
		})
	}
}

func TestGroupClient_GetDescendants(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		tokenSource OrgTokenSource
		data        *GitHubData
		groupID     string
		want        []*groupsync.User
		wantErr     string
	}{
		{
			name: "success",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data: &GitHubData{
				users: map[string]*github.User{
					"user1": {
						ID:    proto.Int64(2286),
						Login: proto.String("user1"),
						Email: proto.String("user1@example.com"),
					},
					"user2": {
						ID:    proto.Int64(5660),
						Login: proto.String("user2"),
						Email: proto.String("user2@example.com"),
					},
					"user3": {
						ID:    proto.Int64(3208),
						Login: proto.String("user3"),
						Email: proto.String("user3@example.com"),
					},
				},
				teams: map[string]map[string]*github.Team{
					"8583": { // org1
						"2797": &github.Team{
							ID:   proto.Int64(2797),
							Name: proto.String("team1"),
							Organization: &github.Organization{
								ID:   proto.Int64(8583),
								Name: proto.String("org1"),
							},
						},
						"9350": &github.Team{
							ID:   proto.Int64(9350),
							Name: proto.String("team2"),
							Organization: &github.Organization{
								ID:   proto.Int64(8583),
								Name: proto.String("org1"),
							},
						},
					},
					"4701": { // org2
						"3387": &github.Team{
							ID:   proto.Int64(3387),
							Name: proto.String("team3"),
							Organization: &github.Organization{
								ID:   proto.Int64(4701),
								Name: proto.String("org2"),
							},
						},
					},
				},
				teamMembers: map[string]map[string]map[string]struct{}{
					"8583": { // org1
						"2797": {
							"user2": struct{}{},
						},
						"9350": {
							"user1": struct{}{},
							"user3": struct{}{},
						},
					},
					"4701": { // org2
						"3387": {
							"user1": struct{}{},
						},
					},
				},
			},
			groupID: "8583:9350",
			want: []*groupsync.User{
				{
					ID: "user1",
					Attributes: &github.User{
						ID:    proto.Int64(2286),
						Login: proto.String("user1"),
						Email: proto.String("user1@example.com"),
					},
				},
				{
					ID: "user3",
					Attributes: &github.User{
						ID:    proto.Int64(3208),
						Login: proto.String("user3"),
						Email: proto.String("user3@example.com"),
					},
				},
			},
		},
		{
			name: "id_wrong_format",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data:    &GitHubData{},
			groupID: "invalidID",
			wantErr: "could not parse groupID invalidID",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			server := fakeGitHub(tc.data)
			defer server.Close()

			client := githubClient(server)

			groupRW := NewGroupReadWriter(tc.tokenSource, client)

			got, err := groupRW.Descendants(ctx, tc.groupID)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error : %v", err)
			}

			// sort so we have a consistent ordering for comparison
			slices.SortFunc(got, func(a, b *groupsync.User) int {
				return strings.Compare(a.ID, b.ID)
			})

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("unexpected gotMembers (-got, +want) = %v", diff)
			}
		})
	}
}

func TestGroupClient_GetUser(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		tokenSource OrgTokenSource
		data        *GitHubData
		userID      string
		want        *groupsync.User
		wantErr     string
	}{
		{
			name: "success",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data: &GitHubData{
				users: map[string]*github.User{
					"user1": {
						ID:    proto.Int64(2286),
						Login: proto.String("user1"),
						Email: proto.String("user1@example.com"),
					},
					"user2": {
						ID:    proto.Int64(5660),
						Login: proto.String("user2"),
						Email: proto.String("user2@example.com"),
					},
					"user3": {
						ID:    proto.Int64(3208),
						Login: proto.String("user3"),
						Email: proto.String("user3@example.com"),
					},
				},
			},
			userID: "user1",
			want: &groupsync.User{
				ID: "user1",
				Attributes: &github.User{
					ID:    proto.Int64(2286),
					Login: proto.String("user1"),
					Email: proto.String("user1@example.com"),
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			server := fakeGitHub(tc.data)
			defer server.Close()

			client := githubClient(server)

			groupRW := NewGroupReadWriter(tc.tokenSource, client)

			got, err := groupRW.GetUser(ctx, tc.userID)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error : %v", err)
			}

			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("unexpected gotMembers (-got, +want) = %v", diff)
			}
		})
	}
}

func TestGroupClient_SetMembers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		tokenSource OrgTokenSource
		data        *GitHubData
		groupID     string
		wantMembers []groupsync.Member
		wantErr     string
	}{
		{
			name: "success_add",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data: &GitHubData{
				users: map[string]*github.User{
					"user1": {
						ID:    proto.Int64(2286),
						Login: proto.String("user1"),
						Email: proto.String("user1@example.com"),
					},
					"user2": {
						ID:    proto.Int64(5660),
						Login: proto.String("user2"),
						Email: proto.String("user2@example.com"),
					},
					"user3": {
						ID:    proto.Int64(3208),
						Login: proto.String("user3"),
						Email: proto.String("user3@example.com"),
					},
				},
				teams: map[string]map[string]*github.Team{
					"8583": { // org1
						"2797": &github.Team{
							ID:   proto.Int64(2797),
							Name: proto.String("team1"),
							Organization: &github.Organization{
								ID:   proto.Int64(8583),
								Name: proto.String("org1"),
							},
						},
						"9350": &github.Team{
							ID:   proto.Int64(9350),
							Name: proto.String("team2"),
							Organization: &github.Organization{
								ID:   proto.Int64(8583),
								Name: proto.String("org1"),
							},
						},
					},
					"4701": { // org2
						"3387": &github.Team{
							ID:   proto.Int64(3387),
							Name: proto.String("team3"),
							Organization: &github.Organization{
								ID:   proto.Int64(4701),
								Name: proto.String("org2"),
							},
						},
					},
				},
				teamMembers: map[string]map[string]map[string]struct{}{
					"8583": { // org1
						"2797": {
							"user2": struct{}{},
						},
						"9350": {
							"user1": struct{}{},
							"user3": struct{}{},
						},
					},
					"4701": { // org2
						"3387": {
							"user1": struct{}{},
						},
					},
				},
			},
			groupID: "8583:2797",
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:    proto.Int64(2286),
							Login: proto.String("user1"),
							Email: proto.String("user1@example.com"),
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user2",
						Attributes: &github.User{
							ID:    proto.Int64(5660),
							Login: proto.String("user2"),
							Email: proto.String("user2@example.com"),
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user3",
						Attributes: &github.User{
							ID:    proto.Int64(3208),
							Login: proto.String("user3"),
							Email: proto.String("user3@example.com"),
						},
					},
				},
			},
		},
		{
			name: "success_remove",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data: &GitHubData{
				users: map[string]*github.User{
					"user1": {
						ID:    proto.Int64(2286),
						Login: proto.String("user1"),
						Email: proto.String("user1@example.com"),
					},
					"user2": {
						ID:    proto.Int64(5660),
						Login: proto.String("user2"),
						Email: proto.String("user2@example.com"),
					},
					"user3": {
						ID:    proto.Int64(3208),
						Login: proto.String("user3"),
						Email: proto.String("user3@example.com"),
					},
				},
				teams: map[string]map[string]*github.Team{
					"8583": { // org1
						"2797": &github.Team{
							ID:   proto.Int64(2797),
							Name: proto.String("team1"),
							Organization: &github.Organization{
								ID:   proto.Int64(8583),
								Name: proto.String("org1"),
							},
						},
						"9350": &github.Team{
							ID:   proto.Int64(9350),
							Name: proto.String("team2"),
							Organization: &github.Organization{
								ID:   proto.Int64(8583),
								Name: proto.String("org1"),
							},
						},
					},
					"4701": { // org2
						"3387": &github.Team{
							ID:   proto.Int64(3387),
							Name: proto.String("team3"),
							Organization: &github.Organization{
								ID:   proto.Int64(4701),
								Name: proto.String("org2"),
							},
						},
					},
				},
				teamMembers: map[string]map[string]map[string]struct{}{
					"8583": { // org1
						"2797": {
							"user2": struct{}{},
						},
						"9350": {
							"user1": struct{}{},
							"user3": struct{}{},
						},
					},
					"4701": { // org2
						"3387": {
							"user1": struct{}{},
						},
					},
				},
			},
			groupID: "8583:9350",
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:    proto.Int64(2286),
							Login: proto.String("user1"),
							Email: proto.String("user1@example.com"),
						},
					},
				},
			},
		},
		{
			name: "success_add_and_remove",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data: &GitHubData{
				users: map[string]*github.User{
					"user1": {
						ID:    proto.Int64(2286),
						Login: proto.String("user1"),
						Email: proto.String("user1@example.com"),
					},
					"user2": {
						ID:    proto.Int64(5660),
						Login: proto.String("user2"),
						Email: proto.String("user2@example.com"),
					},
					"user3": {
						ID:    proto.Int64(3208),
						Login: proto.String("user3"),
						Email: proto.String("user3@example.com"),
					},
				},
				teams: map[string]map[string]*github.Team{
					"8583": { // org1
						"2797": &github.Team{
							ID:   proto.Int64(2797),
							Name: proto.String("team1"),
							Organization: &github.Organization{
								ID:   proto.Int64(8583),
								Name: proto.String("org1"),
							},
						},
						"9350": &github.Team{
							ID:   proto.Int64(9350),
							Name: proto.String("team2"),
							Organization: &github.Organization{
								ID:   proto.Int64(8583),
								Name: proto.String("org1"),
							},
						},
					},
					"4701": { // org2
						"3387": &github.Team{
							ID:   proto.Int64(3387),
							Name: proto.String("team3"),
							Organization: &github.Organization{
								ID:   proto.Int64(4701),
								Name: proto.String("org2"),
							},
						},
					},
				},
				teamMembers: map[string]map[string]map[string]struct{}{
					"8583": { // org1
						"2797": {
							"user2": struct{}{},
						},
						"9350": {
							"user1": struct{}{},
							"user3": struct{}{},
						},
					},
					"4701": { // org2
						"3387": {
							"user1": struct{}{},
						},
					},
				},
			},
			groupID: "8583:9350",
			wantMembers: []groupsync.Member{
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user1",
						Attributes: &github.User{
							ID:    proto.Int64(2286),
							Login: proto.String("user1"),
							Email: proto.String("user1@example.com"),
						},
					},
				},
				&groupsync.UserMember{
					Usr: &groupsync.User{
						ID: "user2",
						Attributes: &github.User{
							ID:    proto.Int64(5660),
							Login: proto.String("user2"),
							Email: proto.String("user2@example.com"),
						},
					},
				},
			},
		},
		{
			name: "id_wrong_format",
			tokenSource: &fakeTokenSource{
				orgTokens: map[int64]string{
					8583: "org_1_test_token",
					4701: "org_2_test_token",
				},
			},
			data:    &GitHubData{},
			groupID: "invalidID",
			wantErr: "could not parse groupID invalidID",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			server := fakeGitHub(tc.data)
			defer server.Close()

			client := githubClient(server)

			groupRW := NewGroupReadWriter(tc.tokenSource, client)

			err := groupRW.SetMembers(ctx, tc.groupID, tc.wantMembers)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error (-got, +want) = %v", diff)
			}

			gotMembers, err := groupRW.GetMembers(ctx, tc.groupID)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error : %v", err)
			}

			// sort so we have a consistent ordering for comparison
			sortByID(gotMembers)

			if diff := cmp.Diff(gotMembers, tc.wantMembers); diff != "" {
				t.Errorf("unexpected gotMembers (-got, +want) = %v", diff)
			}
		})
	}
}

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
		w.Write(jsn)
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
		w.Write(jsn)
	}))
	mux.Handle("GET /organizations/{org_id}/team/{team_id}/members", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(500)
			fmt.Fprintf(w, "missing or malformed authorization header")
			return
		}
		orgID := r.PathValue("org_id")
		team_slug := r.PathValue("team_id")
		teamMembers, ok := githubData.teamMembers[orgID]
		if !ok {
			w.WriteHeader(404)
			fmt.Fprintf(w, "orgID not found")
			return
		}
		members, ok := teamMembers[team_slug]
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
		w.Write(jsn)
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
		username := r.PathValue("username")
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
		w.Write(jsn)
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
		username := r.PathValue("username")
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
	return httptest.NewServer(mux)
}

func sortByID(members []groupsync.Member) {
	slices.SortFunc(members, func(a, b groupsync.Member) int {
		if a.IsUser() {
			userA, _ := a.User()
			if b.IsUser() {
				userB, _ := b.User()
				return strings.Compare(userA.ID, userB.ID)
			} else {
				groupB, _ := b.Group()
				return strings.Compare(userA.ID, groupB.ID)
			}
		} else {
			groupA, _ := a.Group()
			if b.IsUser() {
				userB, _ := b.User()
				return strings.Compare(groupA.ID, userB.ID)
			} else {
				groupB, _ := b.Group()
				return strings.Compare(groupA.ID, groupB.ID)
			}
		}

	})
}
