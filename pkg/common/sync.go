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
	"context"
	"errors"
	"fmt"

	"github.com/abcxyz/team-link/pkg/groupsync"
	"github.com/abcxyz/team-link/pkg/utils"
)

// Sync syncs membership informations.
func Sync(ctx context.Context, mappingFile, configFile string) error {
	var merr error
	mappings, err := utils.ParseMappingTextProto(ctx, mappingFile)
	if err != nil {
		merr = errors.Join(merr, fmt.Errorf("failed to parse mappings file: %w", err))
	}
	config, err := utils.ParseConfigTextProto(ctx, configFile)
	if err != nil {
		merr = errors.Join(merr, fmt.Errorf("failed to parse config file: %w", err))
	}

	if merr != nil {
		return merr
	}

	sourceSystem, targetSystem, err := utils.GetSrcTargetSystemType(config)
	if err != nil {
		return fmt.Errorf("failed to get source and target system type: %w", err)
	}

	srcMapper, targetMapper, err := NewBidirectionalOneToManyGroupMapper(sourceSystem, targetSystem, mappings.GetGroupMappings(), config)
	if err != nil {
		return fmt.Errorf("failed to create mapper: %w", err)
	}

	reader, err := NewReader(ctx, sourceSystem, config)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}

	writer, err := NewReadWriter(ctx, targetSystem, config)
	if err != nil {
		return fmt.Errorf("failed to create writer: %w", err)
	}

	userMapper, err := NewUserMapper(ctx, sourceSystem, targetSystem, mappings.GetUserMappings())
	if err != nil {
		return fmt.Errorf("failed to create user mapper")
	}

	syncer := groupsync.NewManyToManySyncer(sourceSystem, targetSystem, reader, writer, srcMapper, targetMapper, userMapper)
	if err := syncer.SyncAll(ctx); err != nil {
		return fmt.Errorf("failed to sync membership: %w", err)
	}
	return nil
}
