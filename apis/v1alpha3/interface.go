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

package v1alpha3

import "context"

// GroupSyncer syncs groups from a source system to a target system.
type GroupSyncer interface {
	// Name provides descriptive name or identifier of the GroupSyncer
	// implementation. It will be used for logging purpose.
	Name() string

	// SourceSystem provides the name of the source group system.
	SourceSystem() string

	// TargetSystem provides the name of the target group system.
	TargetSystem() string

	// Sync sync the source group with the given ID to the target group system.
	Sync(ctx context.Context, sourceGroupID string) error

	// SyncAll syncs all source groups that this GroupSyncer is aware of to the target system.
	SyncAll(ctx context.Context) error
}
