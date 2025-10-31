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

package groupsync

import (
	"context"
	"errors"
	"fmt"

	"github.com/abcxyz/pkg/logging"
)

// OneToOneSyncer adheres to the v1alpha3.GroupSyncer interface.
// This syncer allows for syncing one source group one target group.
// It adheres to the following policy when syncing a source group ID:
//
//  1. Find the mapped target group of the source group.
//  2. Find all descendants of the source group.
//  3. This set of source descendants is then mapped to their corresponding
//     target users forming the target member set.
//  4. The target member set is then synced to the target group.
type OneToOneSyncer struct {
	name              string // A descriptiven name or identifier for the syncer.
	sourceSystem      string
	targetSystem      string
	sourceGroupReader GroupReader
	targetGroupWriter GroupWriter
	sourceGroupMapper OneToOneGroupMapper
	userMapper        UserMapper
}

type OneToOneSyncerParams struct {
	Name              string
	SourceSystem      string
	TargetSystem      string
	SourceGroupReader GroupReader
	TargetGroupWriter GroupWriter
	SourceGroupMapper OneToOneGroupMapper
	UserMapper        UserMapper
}

// NewOneToOneSyncer creates a new OneToOneSyncer.
func NewOneToOneSyncer(params *OneToOneSyncerParams) *OneToOneSyncer {
	return &OneToOneSyncer{
		name:              params.Name,
		sourceSystem:      params.SourceSystem,
		targetSystem:      params.TargetSystem,
		sourceGroupReader: params.SourceGroupReader,
		targetGroupWriter: params.TargetGroupWriter,
		sourceGroupMapper: params.SourceGroupMapper,
		userMapper:        params.UserMapper,
	}
}

// SourceSystem returns the name of the source group system.
func (f *OneToOneSyncer) SourceSystem() string {
	return f.sourceSystem
}

// TargetSystem returns the name of the target group system.
func (f *OneToOneSyncer) TargetSystem() string {
	return f.targetSystem
}

// Name returns the syncer name.
func (f *OneToOneSyncer) Name() string {
	return f.name
}

// Sync syncs the source group with the given ID to the target group system.
func (f *OneToOneSyncer) Sync(ctx context.Context, sourceGroupID string) error {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "starting sync", "source_group_id", sourceGroupID)

	// Get target group ID for this source group ID
	targetGroup, err := f.sourceGroupMapper.Mapping(ctx, sourceGroupID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to map source group id to target group id",
			"source_group_id", sourceGroupID,
			"error", err,
		)
		return fmt.Errorf("error fetching target group id: %s, %w", sourceGroupID, err)
	}
	logger.InfoContext(ctx, "found the following target group id to sync",
		"source_group_id", sourceGroupID,
		"target_group", targetGroup,
	)

	// Get the union of all users that are members of the source group
	sourceUsers, err := f.sourceUsers(ctx, sourceGroupID)
	if err != nil {
		return fmt.Errorf("error getting source users for source group %s: %w", sourceGroupID, err)
	}
	logger.InfoContext(ctx, "found descendants for source group id",
		"source_users", sourceUsers,
	)

	// Map each source user to their corresponding target user
	targetUsers, err := f.targetUsers(ctx, sourceUsers)
	if err != nil {
		logger.ErrorContext(ctx, "failed mapping one or more source users to their target user",
			"source_users", sourceUsers,
			"target_users", targetUsers,
			"error", err,
		)
		return fmt.Errorf("error getting one or more target users: %w", err)
	}
	logger.InfoContext(ctx, "mapped source users to target users",
		"source_users", sourceUsers,
		"target_users", targetUsers,
	)

	// Map each targetUser to Member type.
	targetMembers := make([]Member, 0, len(targetUsers))
	for _, user := range targetUsers {
		targetMembers = append(targetMembers, &UserMember{Usr: user})
	}

	// targetMembers is now the canonical set of members for the target group ID.
	// Set the target group's members to targetMembers.
	logger.InfoContext(ctx, "setting target group id members to target users",
		"target_group", targetGroup,
		"target_users", targetUsers,
	)
	if err := f.targetGroupWriter.SetMembers(ctx, targetGroup.GroupID, targetMembers); err != nil {
		logger.ErrorContext(ctx, "failed setting target group members",
			"target_group", targetGroup,
			"error", err,
		)
		return fmt.Errorf("error setting members to target group %s: %w", targetGroup, err)
	}

	return nil
}

// SyncAll syncs all source groups that this GroupSyncer is aware of to the target system.
func (f *OneToOneSyncer) SyncAll(ctx context.Context) error {
	sourceGroupIDs, err := f.sourceGroupMapper.AllGroupIDs(ctx)
	if err != nil {
		return fmt.Errorf("error fetching source group ids: %w", err)
	}
	if err := ConcurrentSync(ctx, f, sourceGroupIDs); err != nil {
		return fmt.Errorf("failed to sync one or more ids: %w", err)
	}
	return nil
}

// returns an empty list if none were found.
func (f *OneToOneSyncer) sourceUsers(ctx context.Context, sourceGroupID string) ([]*User, error) {
	sourceUsers, err := f.sourceGroupReader.Descendants(ctx, sourceGroupID)
	if err != nil {
		return nil, fmt.Errorf("error fetching source group descendants: %s, %w", sourceGroupID, err)
	}
	return sourceUsers, nil
}

// returns an empty list if none were found.
func (f *OneToOneSyncer) targetUsers(ctx context.Context, sourceUsers []*User) ([]*User, error) {
	logger := logging.FromContext(ctx)
	targetUsers := make([]*User, 0, len(sourceUsers))
	for _, sourceUser := range sourceUsers {
		targetUser, err := f.userMapper.MappedUser(ctx, sourceUser)
		if errors.Is(err, ErrTargetUserIDNotFound) {
			// if there is no mapping for the target user we will just skip them.
			// it happens when the user is removed from the source system or not mapped to target system.
			logger.DebugContext(ctx, "target user id not found, skipping",
				"source_user", sourceUser,
			)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("error mapping source user id %s to target user id: %w", sourceUser, err)
		}
		targetUsers = append(targetUsers, targetUser)
	}
	return targetUsers, nil
}
