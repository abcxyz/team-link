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

package googlegroupgithub

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	api "github.com/abcxyz/team-link/apis/v1alpha3/proto"
)

func TestCreateBidirectionalGroupMapper(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                          string
		mappings                      *api.GroupMappings
		wantGoogleGroupToGitHubMapper *GroupMapper
		wantGitHubToGoogleGroupMapper *GroupMapper
		wantErr                       string
	}{
		{
			name: "success",
			mappings: &api.GroupMappings{
				Mappings: []*api.GroupMapping{
					{
						Source: &api.GroupMapping_GoogleGroups{
							GoogleGroups: &api.GoogleGroups{
								GroupId: "foo",
							},
						},
						Target: &api.GroupMapping_Github{
							Github: &api.GitHub{
								OrgId:  1,
								TeamId: 2,
							},
						},
					},
					{
						Source: &api.GroupMapping_GoogleGroups{
							GoogleGroups: &api.GoogleGroups{
								GroupId: "foo",
							},
						},
						Target: &api.GroupMapping_Github{
							Github: &api.GitHub{
								OrgId:  1,
								TeamId: 3,
							},
						},
					},
					{
						Source: &api.GroupMapping_GoogleGroups{
							GoogleGroups: &api.GoogleGroups{
								GroupId: "bar",
							},
						},
						Target: &api.GroupMapping_Github{
							Github: &api.GitHub{
								OrgId:  1,
								TeamId: 4,
							},
						},
					},
					{
						Source: &api.GroupMapping_GoogleGroups{
							GoogleGroups: &api.GoogleGroups{
								GroupId: "foobar",
							},
						},
						Target: &api.GroupMapping_Github{
							Github: &api.GitHub{
								OrgId:  1,
								TeamId: 4,
							},
						},
					},
				},
			},
			wantGoogleGroupToGitHubMapper: &GroupMapper{
				mappings: map[string][]string{
					"foo":    {"1:2", "1:3"},
					"bar":    {"1:4"},
					"foobar": {"1:4"},
				},
			},
			wantGitHubToGoogleGroupMapper: &GroupMapper{
				mappings: map[string][]string{
					"1:2": {"foo"},
					"1:3": {"foo"},
					"1:4": {"bar", "foobar"},
				},
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			BiDirectionalGroupMapper := NewBidirectionaGroupMapper(tc.mappings)
			if diff := cmp.Diff(BiDirectionalGroupMapper.SourceMapper.mappings, tc.wantGoogleGroupToGitHubMapper.mappings, cmp.AllowUnexported()); diff != "" {
				t.Errorf("got unexpected GoogleGroupToGitHubMapper:\n%s", diff)
			}
			if diff := cmp.Diff(BiDirectionalGroupMapper.TargetMapper.mappings, tc.wantGitHubToGoogleGroupMapper.mappings, cmp.AllowUnexported()); diff != "" {
				t.Errorf("got unexpected GitHubToGoogleGroupMapper:\n%s", diff)
			}
		})
	}
}

func TestNewUserMapper(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                              string
		mappings                          *api.UserMappings
		wantGoogleGroupToGitHubUserMapper *GoogleGroupGitHubUserMapper
		wantErr                           string
	}{
		{
			name: "success",
			mappings: &api.UserMappings{
				Mappings: []*api.UserMapping{
					{
						Source: "src_id_1",
						Target: "target_id_1",
					},
					{
						Source: "src_id_2",
						Target: "target_id_2",
					},
				},
			},
			wantGoogleGroupToGitHubUserMapper: &GoogleGroupGitHubUserMapper{
				mappings: map[string]string{
					"src_id_1": "target_id_1",
					"src_id_2": "target_id_2",
				},
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			gotGGToGH := NewUserMapper(ctx, tc.mappings)
			if diff := cmp.Diff(gotGGToGH.mappings, tc.wantGoogleGroupToGitHubUserMapper.mappings, cmp.AllowUnexported()); diff != "" {
				t.Errorf("got unexpected GoogleGroupToGitHubMapper:\n%s", diff)
			}
		})
	}
}
