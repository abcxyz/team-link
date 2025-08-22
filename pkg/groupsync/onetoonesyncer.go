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

// NewOneToOneSyncer creates a new OneToOneSyncer.
func NewOneToOneSyncer(
	name string,
	sourceSystem string,
	targetSystem string,
	sourceGroupClient GroupReader,
	targetGroupClient GroupWriter,
	sourceGroupMapper OneToOneGroupMapper,
	userMapper UserMapper,
) *OneToOneSyncer {
	return &OneToOneSyncer{
		name:              name,
		sourceSystem:      sourceSystem,
		targetSystem:      targetSystem,
		sourceGroupReader: sourceGroupClient,
		targetGroupWriter: targetGroupClient,
		sourceGroupMapper: sourceGroupMapper,
		userMapper:        userMapper,
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
	targetGroupID, err := f.sourceGroupMapper.MappedGroupID(ctx, sourceGroupID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to map source group id to target group id",
			"source_group_id", sourceGroupID,
			"error", err,
		)
		return fmt.Errorf("error fetching target group id: %s, %w", sourceGroupID, err)
	}
	logger.InfoContext(ctx, "found the following target group id to sync",
		"source_group_id", sourceGroupID,
		"target_group_id", targetGroupID,
	)

	var merr error
	// Get the union of all users that are members of the source group
	sourceUsers, err := f.sourceUsers(ctx, sourceGroupID)
	sourceUserIds := userIDs(sourceUsers)
	if err != nil {
		return fmt.Errorf("error getting source users for source group %s: %w", sourceGroupID, err)
	}
	logger.DebugContext(ctx, "found descendants for source group id",
		"source_user_ids", sourceUserIds,
	)

	// Map each source user to their corresponding target user
	targetUsers, err := f.targetUsers(ctx, sourceUsers)
	targetUserIds := userIDs(targetUsers)
	if err != nil {
		logger.ErrorContext(ctx, "failed mapping one or more source users to their target user",
			"source_user_ids", sourceUserIds,
			"target_user_ids", targetUserIds,
			"error", err,
		)
		merr = errors.Join(merr, fmt.Errorf("error getting one or more target users: %w", err))
	}
	logger.DebugContext(ctx, "mapped source users to target users",
		"source_user_ids", sourceUserIds,
		"target_user_ids", targetUserIds,
	)

	// map each targetUser to Member type
	targetMembers := make([]Member, 0, len(targetUsers))
	for _, user := range targetUsers {
		targetMembers = append(targetMembers, &UserMember{Usr: user})
	}

	// targetMembers is now the canonical set of members for the target group ID.
	// Set the target group's members to targetMembers.
	logger.DebugContext(ctx, "setting target group id members to target users",
		"target_group_id", targetGroupID,
		"target_user_ids", targetUserIds,
	)
	if err := f.targetGroupWriter.SetMembers(ctx, targetGroupID, targetMembers); err != nil {
		logger.ErrorContext(ctx, "failed setting target group members",
			"target_group_id", targetGroupID,
			"error", err,
		)
		merr = fmt.Errorf("error setting members to target group %s: %w", targetGroupID, err)
	}

	return merr
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
	var merr error
	targetUsers := make([]*User, 0, len(sourceUsers))
	for _, sourceUser := range sourceUsers {
		targetUser, err := f.userMapper.MappedUser(ctx, sourceUser)
		if errors.Is(err, ErrTargetUserIDNotFound) {
			logger.DebugContext(ctx, "target user id not found, skipping",
				"source_user_id", sourceUser.ID,
			)
			continue
		}
		if err != nil {
			merr = errors.Join(merr, fmt.Errorf("error mapping source user id %s to target user id: %w", sourceUser.ID, err))
			continue
		}
		targetUsers = append(targetUsers, targetUser)
	}
	return targetUsers, merr
}
