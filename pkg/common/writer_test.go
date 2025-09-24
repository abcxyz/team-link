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

package common

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	api "github.com/abcxyz/team-link/v2/apis/v1alpha3/proto"
)

func TestComputeOrgTeamSSORequired(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                   string
		mappings               *api.TeamLinkMappings
		wantOrgTeamSSORequired map[int64]map[int64]bool
	}{
		{
			name: "success_single_org_single_team",
			mappings: &api.TeamLinkMappings{
				GroupMappings: &api.GroupMappings{
					Mappings: []*api.GroupMapping{
						{
							Source: nil,
							Target: &api.GroupMapping_Github{
								Github: &api.GitHub{
									OrgId:                1,
									TeamId:               1,
									RequireUserEnableSso: true,
								},
							},
						},
					},
				},
			},
			wantOrgTeamSSORequired: map[int64]map[int64]bool{
				1: {
					1: true,
				},
			},
		},
		{
			name: "success_multiple_org_multiple_team",
			mappings: &api.TeamLinkMappings{
				GroupMappings: &api.GroupMappings{
					Mappings: []*api.GroupMapping{
						{
							Source: nil,
							Target: &api.GroupMapping_Github{
								Github: &api.GitHub{
									OrgId:                1,
									TeamId:               1,
									RequireUserEnableSso: true,
								},
							},
						},
						{
							Source: nil,
							Target: &api.GroupMapping_Github{
								Github: &api.GitHub{
									OrgId:                1,
									TeamId:               2,
									RequireUserEnableSso: false,
								},
							},
						},
						{
							Source: nil,
							Target: &api.GroupMapping_Github{
								Github: &api.GitHub{
									OrgId:                2,
									TeamId:               1,
									RequireUserEnableSso: true,
								},
							},
						},
						{
							Source: nil,
							Target: &api.GroupMapping_Github{
								Github: &api.GitHub{
									OrgId:                2,
									TeamId:               2,
									RequireUserEnableSso: false,
								},
							},
						},
					},
				},
			},
			wantOrgTeamSSORequired: map[int64]map[int64]bool{
				1: {
					1: true,
					2: false,
				},
				2: {
					1: true,
					2: false,
				},
			},
		},
		{
			name: "success_multiple_org_multiple_team_with_default_sso",
			mappings: &api.TeamLinkMappings{
				GroupMappings: &api.GroupMappings{
					Mappings: []*api.GroupMapping{
						{
							Source: nil,
							Target: &api.GroupMapping_Github{
								Github: &api.GitHub{
									OrgId:                1,
									TeamId:               1,
									RequireUserEnableSso: true,
								},
							},
						},
						{
							Source: nil,
							Target: &api.GroupMapping_Github{
								Github: &api.GitHub{
									OrgId:  1,
									TeamId: 2,
								},
							},
						},
						{
							Source: nil,
							Target: &api.GroupMapping_Github{
								Github: &api.GitHub{
									OrgId:                2,
									TeamId:               1,
									RequireUserEnableSso: true,
								},
							},
						},
						{
							Source: nil,
							Target: &api.GroupMapping_Github{
								Github: &api.GitHub{
									OrgId:  2,
									TeamId: 2,
								},
							},
						},
					},
				},
			},
			wantOrgTeamSSORequired: map[int64]map[int64]bool{
				1: {
					1: true,
					2: false,
				},
				2: {
					1: true,
					2: false,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotOrgTeamSSORequired := computeOrgTeamSSORequired(tc.mappings)
			if diff := cmp.Diff(gotOrgTeamSSORequired, tc.wantOrgTeamSSORequired); diff != "" {
				t.Errorf("got unexpected OrgTeamSSORequired:\n%s", diff)
			}
		})
	}
}
