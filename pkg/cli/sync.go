// Copyright 2022 Google LLC
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

package cli

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/abcxyz/pkg/cli"
	tltypes "github.com/abcxyz/team-link/internal"
	"github.com/abcxyz/team-link/pkg/client"
	"github.com/abcxyz/team-link/pkg/groupsync"
)

var (
	_                        cli.Command = (*SyncCommand)(nil)
	allowedSourceSystem                  = []string{tltypes.SystemTypeGoogleGroups}
	allowedDestinationSystem             = []string{tltypes.SystemTypeGitHub}
)

type SyncCommand struct {
	cli.BaseCommand

	source             string
	destination        string
	groupMappingConfig string
	userMappingConfig  string
	clientConfig       client.ClientConfig
}

func (c *SyncCommand) Desc() string {
	return `Sync membership information`
}

func (c *SyncCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]

  Sync membership information from source and target system.

  Allowed source system: GoogleGroups.

  Allowed destination system: GitHub.

  For example, to sync membership from google group to GitHub.

  tlctl sync run \
	-src GoogleGroup \
	-dst GitHub \
	-group-mapping-config your-group-config.textproto \
	-user-mapping-config your-user-confif.textproto \
	-github-client-auth-token you-token
`
}

func (c *SyncCommand) RegisterFlags(set *cli.FlagSet) {
	f := set.NewSection("COMMAND OPTIONS")
	f.StringVar(&cli.StringVar{
		Name:    "source",
		Target:  &c.source,
		Aliases: []string{"src", "s"},
		Example: "GoogleGroup",
		Usage:   `The source system to read membership `,
	})

	f.StringVar(&cli.StringVar{
		Name:    "destination",
		Target:  &c.destination,
		Aliases: []string{"d", "dst"},
		Example: "GitHub",
		Usage:   `The target system for membership information`,
	})

	f.StringVar(&cli.StringVar{
		Name:    "group-mapping-config",
		Target:  &c.groupMappingConfig,
		Aliases: []string{"gc"},
		Example: "group-mapping-config.textproto",
		Usage: `The group mapping config that contains group id mapping ` +
			`from source system to destination system`,
	})

	f.StringVar(&cli.StringVar{
		Name:    "user-mapping-config",
		Target:  &c.userMappingConfig,
		Aliases: []string{"uc"},
		Example: "user-mapping-config.textproto",
		Usage: `The group mapping config that contains group id mapping ` +
			`from source system to destination system`,
	})

	set.AfterParse(func(merr error) error {
		// Convert source and destination to all caps so it matches the
		// predefined system name const.
		c.source = strings.ToUpper(c.source)
		c.destination = strings.ToUpper(c.destination)

		if ok := slices.Contains(allowedSourceSystem, c.source); !ok {
			merr = errors.Join(merr, fmt.Errorf("source system %s not in allowed list: %s", c.source, strings.Join(allowedSourceSystem, ",")))
		}

		if ok := slices.Contains(allowedDestinationSystem, c.destination); !ok {
			merr = errors.Join(merr, fmt.Errorf("destination system %s not in allowed list: %s", c.destination, strings.Join(allowedDestinationSystem, ",")))
		}

		if c.groupMappingConfig == "" {
			merr = errors.Join(merr, fmt.Errorf("group mapping config file is not provided"))
		}

		if c.userMappingConfig == "" {
			merr = errors.Join(merr, fmt.Errorf("user mapping config file is not provided"))
		}

		if c.destination == tltypes.SystemTypeGitHub && c.clientConfig.GitHub.Token == "" {
			merr = errors.Join(merr, fmt.Errorf("auth token not provided for destination system %s", c.destination))
		}
		return merr
	})
}

func (c *SyncCommand) Flags() *cli.FlagSet {
	set := c.NewFlagSet()

	c.clientConfig.RegisterFlags(set)
	c.RegisterFlags(set)

	return set
}

func (c *SyncCommand) Run(ctx context.Context, args []string) error {
	f := c.Flags()
	if err := f.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	args = f.Args()
	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %q", args)
	}

	if ok := slices.Contains(allowedSourceSystem, c.source); !ok {
		return fmt.Errorf("source system %s not in allowed list: %s", c.source, strings.Join(allowedSourceSystem, ","))
	}

	sm, dm, err := client.NewBidirectionalOneToManyGroupMapper(c.source, c.destination, c.groupMappingConfig)
	if err != nil {
		return fmt.Errorf("failed to create group mapper: %w", err)
	}

	um, err := client.NewUserMapper(ctx, c.source, c.destination, c.userMappingConfig)
	if err != nil {
		return fmt.Errorf("failed to create user mapper: %w", err)
	}

	reader, err := client.NewReader(ctx, c.source)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}

	readWriter, err := client.NewReadWriter(ctx, c.destination, &c.clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create readwriter: %w", err)
	}

	syncer := groupsync.NewManyToManySyncer(c.source, c.destination, reader, readWriter, sm, dm, um)
	if err := syncer.SyncAll(ctx); err != nil {
		return fmt.Errorf("failed to sync %w", err)
	}

	return nil
}
