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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v67/github"

	"github.com/abcxyz/pkg/testutil"
)

// EnterpriseUserData holds the state for the mock server.
type EnterpriseUserData struct {
	allUsers            map[string]*github.SCIMUserAttributes // Map SCIM ID to user
	failListUserCalls   bool
	failCreateUserCalls bool
}

// fakeEnterprise returns a test server that mocks the GHES SCIM API.
func fakeEnterprise(t *testing.T, data *EnterpriseUserData) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	basePath := "/api/v3/scim/v2/Users"

	mux.HandleFunc(basePath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet: // List
			if data.failListUserCalls {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			var users []*github.SCIMUserAttributes
			for _, u := range data.allUsers {
				users = append(users, u)
			}
			list := &github.SCIMProvisionedIdentities{Resources: users, TotalResults: github.Int(len(users))}
			if err := json.NewEncoder(w).Encode(list); err != nil {
				t.Fatalf("failed to encode list response: %v", err)
			}
		case http.MethodPost: // Create
			if data.failCreateUserCalls {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			var user github.SCIMUserAttributes
			if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			user.ID = github.String(fmt.Sprintf("scim-id-%s", user.UserName))
			data.allUsers[*user.ID] = &user
			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(user); err != nil {
				t.Fatalf("failed to encode create response: %v", err)
			}
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc(basePath+"/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, basePath+"/")
		switch r.Method {
		case http.MethodGet:
			user, ok := data.allUsers[id]
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			if err := json.NewEncoder(w).Encode(user); err != nil {
				t.Fatalf("failed to encode create response: %v", err)
			}
		case http.MethodPut:
			if _, ok := data.allUsers[id]; !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			var updatedUser github.SCIMUserAttributes
			if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			updatedUser.ID = github.String(id)
			data.allUsers[id] = &updatedUser
			if err := json.NewEncoder(w).Encode(updatedUser); err != nil {
				t.Fatalf("failed to encode create response: %v", err)
			}
		case http.MethodPatch:
			user, ok := data.allUsers[id]
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			var patch scimPatchPayload
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if len(patch.Operations) > 0 {
				op := patch.Operations[0]
				if op.Op == "replace" {
					if value, ok := op.Value.(map[string]interface{}); ok {
						if active, ok := value["active"].(bool); ok {
							user.Active = github.Bool(active)
						}
					}
				}
			}

			if err := json.NewEncoder(w).Encode(user); err != nil {
				t.Fatalf("failed to encode create response: %v", err)
			}
		case http.MethodDelete:
			if _, ok := data.allUsers[id]; !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			delete(data.allUsers, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}

func TestSCIMClient_ListUsers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name              string
		initialUsers      map[string]*github.SCIMUserAttributes
		failListUserCalls bool
		want              map[string]*github.SCIMUserAttributes
		wantErrStr        string
	}{
		{
			name: "success_multiple_users",
			initialUsers: map[string]*github.SCIMUserAttributes{
				"id1": {ID: github.String("id1"), UserName: "user.one"},
				"id2": {ID: github.String("id2"), UserName: "user.two"},
			},
			want: map[string]*github.SCIMUserAttributes{
				"user.one": {ID: github.String("id1"), UserName: "user.one"},
				"user.two": {ID: github.String("id2"), UserName: "user.two"},
			},
		},
		{
			name:         "success_no_users",
			initialUsers: map[string]*github.SCIMUserAttributes{},
			want:         map[string]*github.SCIMUserAttributes{},
		},
		{
			name:              "error_api_fails",
			initialUsers:      map[string]*github.SCIMUserAttributes{},
			failListUserCalls: true,
			wantErrStr:        "request failed with status 500",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			userData := &EnterpriseUserData{allUsers: make(map[string]*github.SCIMUserAttributes), failListUserCalls: tc.failListUserCalls}
			for k, v := range tc.initialUsers {
				userData.allUsers[k] = v
			}
			srv := fakeEnterprise(t, userData)
			defer srv.Close()

			client, err := NewSCIMClient(srv.Client(), srv.URL)
			if err != nil {
				t.Fatalf("NewSCIMClient failed: %v", err)
			}

			got, err := client.ListUsers(ctx)
			if diff := testutil.DiffErrString(err, tc.wantErrStr); diff != "" {
				t.Errorf("error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ListUsers response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSCIMClient_UpdateUser(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		scimIDToUpdate  string
		updatePayload   *github.SCIMUserAttributes
		wantFinalUser   *github.SCIMUserAttributes
		wantServerCount int
		wantErrStr      string
	}{
		{
			name:           "success",
			scimIDToUpdate: "id1",
			updatePayload: &github.SCIMUserAttributes{
				Name: github.SCIMUserName{GivenName: "New", FamilyName: "Name"},
			},
			wantFinalUser: &github.SCIMUserAttributes{
				ID:   github.String("id1"),
				Name: github.SCIMUserName{GivenName: "New", FamilyName: "Name"},
				Schemas: []string{
					"urn:ietf:params:scim:schemas:core:2.0:User",
				},
			},
			wantServerCount: 1,
		},
		{
			name:           "error_not_found",
			scimIDToUpdate: "nonexistent",
			updatePayload: &github.SCIMUserAttributes{
				Name: github.SCIMUserName{GivenName: "New", FamilyName: "Name"},
			},
			wantErrStr:      "request failed with status 404",
			wantServerCount: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			userData := &EnterpriseUserData{allUsers: map[string]*github.SCIMUserAttributes{
				"id1": {ID: github.String("id1"), UserName: "test.user", Name: github.SCIMUserName{GivenName: "Old", FamilyName: "Name"}},
			}}
			srv := fakeEnterprise(t, userData)
			defer srv.Close()

			client, err := NewSCIMClient(srv.Client(), srv.URL)
			if err != nil {
				t.Fatalf("NewSCIMClient failed: %v", err)
			}

			_, _, err = client.UpdateUser(ctx, tc.scimIDToUpdate, tc.updatePayload)
			if diff := testutil.DiffErrString(err, tc.wantErrStr); diff != "" {
				t.Errorf("error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantFinalUser, userData.allUsers[tc.scimIDToUpdate]); diff != "" {
				t.Errorf("user on server mismatch (-want +got):\n%s", diff)
			}
			if len(userData.allUsers) != tc.wantServerCount {
				t.Errorf("server should have %d users, but has %d", tc.wantServerCount, len(userData.allUsers))
			}
		})
	}
}

func TestSCIMClient_DeleteUser(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		scimIDToDelete  string
		wantServerCount int
		wantErrStr      string
	}{
		{
			name:            "success",
			scimIDToDelete:  "id1",
			wantServerCount: 0,
		},
		{
			name:            "error_not_found",
			scimIDToDelete:  "nonexistent",
			wantServerCount: 1,
			wantErrStr:      "request failed with status 404",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			userData := &EnterpriseUserData{allUsers: map[string]*github.SCIMUserAttributes{
				"id1": {ID: github.String("id1"), UserName: "test.user"},
			}}
			srv := fakeEnterprise(t, userData)
			defer srv.Close()

			client, err := NewSCIMClient(srv.Client(), srv.URL)
			if err != nil {
				t.Fatalf("NewSCIMClient failed: %v", err)
			}

			_, err = client.DeleteUser(ctx, tc.scimIDToDelete)
			if diff := testutil.DiffErrString(err, tc.wantErrStr); diff != "" {
				t.Errorf("error mismatch (-want +got):\n%s", diff)
			}
			if len(userData.allUsers) != tc.wantServerCount {
				t.Errorf("server should have %d users, but has %d", tc.wantServerCount, len(userData.allUsers))
			}
		})
	}
}

func TestSCIMClient_CreateUser(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                string
		failToCreateUser    bool
		userToCreate        *github.SCIMUserAttributes
		wantUserOnServer    *github.SCIMUserAttributes
		wantServerUserCount int
		wantErrStr          string
	}{
		{
			name:         "success",
			userToCreate: &github.SCIMUserAttributes{UserName: "new.user"},
			wantUserOnServer: &github.SCIMUserAttributes{
				ID:       github.String("scim-id-new.user"),
				UserName: "new.user",
				Schemas: []string{
					"urn:ietf:params:scim:schemas:core:2.0:User",
				},
				Active: github.Bool(true),
			},
			wantServerUserCount: 1,
		},
		{
			name:                "fail_internal_server_error",
			failToCreateUser:    true,
			userToCreate:        &github.SCIMUserAttributes{UserName: "new.user"},
			wantServerUserCount: 0,
			wantErrStr:          "request failed with status 500",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			userData := &EnterpriseUserData{
				allUsers:            make(map[string]*github.SCIMUserAttributes),
				failCreateUserCalls: tc.failToCreateUser,
			}
			srv := fakeEnterprise(t, userData)
			defer srv.Close()

			client, err := NewSCIMClient(srv.Client(), srv.URL)
			if err != nil {
				t.Fatalf("NewSCIMClient failed: %v", err)
			}

			createdUser, _, err := client.CreateUser(ctx, tc.userToCreate)
			if diff := testutil.DiffErrString(err, tc.wantErrStr); diff != "" {
				t.Errorf("error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantUserOnServer, createdUser); diff != "" {
				t.Errorf("created user mismatch (-want +got):\n%s", diff)
			}
			if len(userData.allUsers) != tc.wantServerUserCount {
				t.Errorf("server should have %d users, but has %d", tc.wantServerUserCount, len(userData.allUsers))
			}
		})
	}
}

func TestSCIMClient_GetUser(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		scimIDToGet string
		wantUser    *github.SCIMUserAttributes
		wantErrStr  string
	}{
		{
			name:        "success_found",
			scimIDToGet: "id1",
			wantUser:    &github.SCIMUserAttributes{ID: github.String("id1"), UserName: "test.user"},
		},
		{
			name:        "error_not_found",
			scimIDToGet: "nonexistent",
			wantErrStr:  "request failed with status 404",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			userData := &EnterpriseUserData{allUsers: map[string]*github.SCIMUserAttributes{
				"id1": {ID: github.String("id1"), UserName: "test.user"},
			}}
			srv := fakeEnterprise(t, userData)
			defer srv.Close()

			client, err := NewSCIMClient(srv.Client(), srv.URL)
			if err != nil {
				t.Fatalf("NewSCIMClient failed: %v", err)
			}

			got, _, err := client.GetUser(ctx, tc.scimIDToGet)
			if diff := testutil.DiffErrString(err, tc.wantErrStr); diff != "" {
				t.Errorf("error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantUser, got); diff != "" {
				t.Errorf("user mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSCIMClient_DeactivateUser(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name               string
		scimIDToDeactivate string
		users              map[string]*github.SCIMUserAttributes
		wantUser           *github.SCIMUserAttributes
		wantErrStr         string
	}{
		{
			name:               "success",
			scimIDToDeactivate: "id1",
			users: map[string]*github.SCIMUserAttributes{
				"id1": {ID: github.String("id1"), UserName: "test.user", Active: github.Bool(true)},
			},
			wantUser: &github.SCIMUserAttributes{
				ID:       github.String("id1"),
				UserName: "test.user",
				Active:   github.Bool(false),
			},
		},
		{
			name:               "error_not_found",
			scimIDToDeactivate: "nonexistent",
			wantErrStr:         "request failed with status 404",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			userData := &EnterpriseUserData{allUsers: tc.users}
			srv := fakeEnterprise(t, userData)
			defer srv.Close()

			client, err := NewSCIMClient(srv.Client(), srv.URL)
			if err != nil {
				t.Fatalf("NewSCIMClient failed: %v", err)
			}

			got, _, err := client.DeactivateUser(ctx, tc.scimIDToDeactivate)
			if diff := testutil.DiffErrString(err, tc.wantErrStr); diff != "" {
				t.Errorf("error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantUser, got); diff != "" {
				t.Errorf("user mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSCIMClient_ReactivateUser(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name               string
		scimIDToReactivate string
		users              map[string]*github.SCIMUserAttributes
		wantUser           *github.SCIMUserAttributes
		wantErrStr         string
	}{
		{
			name:               "success",
			scimIDToReactivate: "id1",
			users: map[string]*github.SCIMUserAttributes{
				"id1": {ID: github.String("id1"), UserName: "test.user", Active: github.Bool(false)},
			},
			wantUser: &github.SCIMUserAttributes{
				ID:       github.String("id1"),
				UserName: "test.user",
				Active:   github.Bool(true),
			},
		},
		{
			name:               "error_not_found",
			scimIDToReactivate: "nonexistent",
			wantErrStr:         "request failed with status 404",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			userData := &EnterpriseUserData{allUsers: tc.users}
			srv := fakeEnterprise(t, userData)
			defer srv.Close()

			client, err := NewSCIMClient(srv.Client(), srv.URL)
			if err != nil {
				t.Fatalf("NewSCIMClient failed: %v", err)
			}

			got, _, err := client.ReactivateUser(ctx, tc.scimIDToReactivate)
			if diff := testutil.DiffErrString(err, tc.wantErrStr); diff != "" {
				t.Errorf("error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantUser, got); diff != "" {
				t.Errorf("user mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
