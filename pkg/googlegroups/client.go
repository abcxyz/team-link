// Copyright 2024 Google LLC
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

package googlegroups

import (
	"context"
	"fmt"

	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/cloudidentity/v1"

	"github.com/abcxyz/team-link/v2/pkg/groupsync"
)

// NewGroupReaderWithDefaultApplicationToken creates a reader for GoogleGroups.
// This uses default auth login token to authenticate. The token is stored in
// environment variable GOOGLE_APPLICATION_CREDENTIALS.
// See:
// https://cloud.google.com/docs/authentication/application-default-credentials
//
// This Envvar will be auto-written if you run command `gcloud auth application-default login`
// or run github action google-gihub-actions/auth.
func NewGroupReaderWithDefaultApplicationToken(ctx context.Context) (groupsync.GroupReader, error) {
	cs, err := cloudidentity.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloudidentity service: %w", err)
	}
	as, err := admin.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin service: %w", err)
	}
	return NewGroupReader(cs, as), nil
}
