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

package teamlink

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/go-github/v61/github"

	"github.com/abcxyz/pkg/githubauth"
	"github.com/abcxyz/pkg/testutil"
	"github.com/abcxyz/team-link/apis/v1alpha2"
	tlgithub "github.com/abcxyz/team-link/pkg/github"
)

const testPrivateKey = `IyBDT05GSVJNRURURVNUS0VZCi0tLS0tQkVHS
U4gUlNBIFBSSVZBVEUgS0VZLS0tLS0KTUlJQ1hBSUJBQUtCZ1FDcUdLdWtPMURlN3poWmo2K0gwcXRqVGtWeHdUQ3B2S2U0ZUNaM
EZQcXJpMGNiMkpaZlhKL0RnWVNGNnZVcAp3bUpHOHdWUVpLamVHY2pET0w1VWxzdXVzRm5jQ3pXQlE3UktOVVNlc21RUk1TR2tWY
jEvM2orc2taNlV0Vys1dTA5bEhOc2o2dFE1CjFzMVNQckNCa2VkYk5mMFRwMEdiTUpEeVI0ZTlUMDRaWndJREFRQUJBb0dBRmlqa
281NitxR3lOOE0wUlZ5YVJBWHorK3hUcUhCTGgKM3R4NFZnTXRyUStXRWdDamhvVHdvMjNLTUJBdUpHU1luUm1vQlpNM2xNZlRLZ
XZJa0FpZFBFeHZZQ2RtNWRZcTNYVG9Ma2tMdjVMMgpwSUlWT0ZNREcrS0VTbkFGVjdsMmMrY256Uk1XMCtiNmY4bVIxQ0p6WnV4V
kxMNlEwMmZ2TGk1NS9tYlNZeEVDUVFEZUF3NmZpSVFYCkd1a0JJNGVNWlp0NG5zY3kybzEyS3lZbmVyM1Zwb2VFK05wMnErWjNwd
kFNZC9hTnpRL1c5V2FJK05SZmN4VUpybWZQd0lHbTYzaWwKQWtFQXhDTDVIUWIyYlFyNEJ5b3JjTVdtL2hFUDJNWnpST1Y3M3lGN
DFoUHNSQzltNjZLcmhlTzlIUFRKdW8zLzlzNXArc3FHeE9sRgpMME5EdDRTa29zamdHd0pBRmtseVIxdVovd1BKamo2MTFjZEJje
nRsUGRxb3hzc1FHbmg4NUJ6Q2ovdTNXcUJwRTJ2anZ5eXZ5STVrClg2ems3UzBsakt0dDJqbnkyKzAwVnNCZXJRSkJBSkdDMU1nN
U95ZG81TndENkJpUk9yUHhHbzJicFRidS9maHJUOGViSGtUejJlcGwKVTlWUVFTUXpZMW9aTVZYOGkxbTVXVVRMUHoyeUxKSUJRV
mRYcWhNQ1FCR29pdVNvU2phZlVoVjdpMWNFR3BiODhoNU5CWVp6V1hHWgozN3NKNVFzVytzSnlvTmRlM3hIOHZkWGh6VTdlVDgyR
DZYL3NjdzlSWnorLzZyQ0o0cDA9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==`

var (
	testPrivateKeyBytes = Must(base64.StdEncoding.DecodeString(testPrivateKey))
	testHTTPClient      = &http.Client{}
	testGitHubClient    = github.NewClient(testHTTPClient)
)

func TestGet(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		key             []byte
		keyErr          error
		cacheDuration   time.Duration
		githubAppID     string
		syncOpts        []tlgithub.Option
		wantErrSubstr   string
		wantAccessCount int
		wantSyncer      v1alpha2.TeamSynchronizer
	}{
		{
			name:            "success",
			key:             testPrivateKeyBytes,
			cacheDuration:   5 * time.Minute,
			githubAppID:     "12345",
			wantAccessCount: 1,
			wantSyncer: Must(tlgithub.NewSynchronizer(
				testGitHubClient,
				Must(githubauth.NewApp(
					"12345",
					string(testPrivateKeyBytes),
					githubauth.WithHTTPClient(testHTTPClient),
				)),
			)),
		},
		{
			name:            "success_with_opts",
			key:             testPrivateKeyBytes,
			cacheDuration:   5 * time.Minute,
			githubAppID:     "12345",
			syncOpts:        []tlgithub.Option{tlgithub.WithDryRun()},
			wantAccessCount: 1,
			wantSyncer: Must(tlgithub.NewSynchronizer(
				testGitHubClient,
				Must(githubauth.NewApp(
					"12345",
					string(testPrivateKeyBytes),
					githubauth.WithHTTPClient(testHTTPClient),
				)),
				tlgithub.WithDryRun(),
			)),
		},
		{
			name:            "failed_create_github_app",
			key:             []byte("invalid private key"),
			cacheDuration:   5 * time.Minute,
			githubAppID:     "12345",
			wantAccessCount: 1,
			wantErrSubstr:   "failed to create github app",
		},
		{
			name:            "failed_secret_not_found",
			cacheDuration:   5 * time.Minute,
			githubAppID:     "12345",
			keyErr:          fmt.Errorf("secret not found"),
			wantAccessCount: 0,
			wantErrSubstr:   "secret not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			staticKeyProvider := &staticKeyProvider{key: tc.key, keyErr: tc.keyErr}
			configuredSyncer := NewConfiguredSyncer(
				staticKeyProvider,
				testHTTPClient,
				testGitHubClient,
				tc.cacheDuration,
				tc.githubAppID,
				tc.syncOpts...,
			)
			syncer, gotErr := configuredSyncer.Get(ctx)
			if diff := testutil.DiffErrString(gotErr, tc.wantErrSubstr); diff != "" {
				t.Errorf("Process(%+v) got unexpected error substring: %v", tc.name, diff)
			}

			opts := []cmp.Option{
				cmp.AllowUnexported(tlgithub.Synchronizer{}),
				// GitHub client will be the same object that is passed in.
				cmpopts.EquateComparable(github.Client{}),
				cmp.AllowUnexported(githubauth.App{}),
				cmpopts.IgnoreFields(githubauth.App{},
					"installationCache",
					"installationCacheLock"),
			}

			if diff := cmp.Diff(tc.wantSyncer, syncer, opts...); diff != "" {
				t.Errorf("Process(%+v) got syncer diff (-want, +got): %v", tc.name, diff)
			}
			// Verify the key access count/if it is using cache.
			_, _ = configuredSyncer.Get(ctx)
			if diff := cmp.Diff(tc.wantAccessCount, staticKeyProvider.accessCount); diff != "" {
				t.Errorf("Process(%+v) got access count (-want, +got): %v", tc.name, diff)
			}
		})
	}
}

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

type staticKeyProvider struct {
	key         []byte
	keyErr      error
	accessCount int
}

func (k *staticKeyProvider) Key(ctx context.Context) ([]byte, error) {
	if k.keyErr != nil {
		return nil, k.keyErr
	}
	k.accessCount++
	return k.key, nil
}
