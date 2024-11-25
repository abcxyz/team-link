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
	"fmt"
	"slices"
	"strings"

	"github.com/abcxyz/pkg/cli"
)

var (
	_                        cli.Command = (*SyncCommand)(nil)
	allowedSourceSystem                  = []string{"GOOGLEGROUPS"}
	allowedDestinationSystem             = []string{"GITHUB"}
)

type SyncCommand struct {
	cli.BaseCommand

	source             string
	destination        string
	groupMappingConfig string
	userMappingConfig  string
	sourceToken        string
	destinationToken   string
}

func (c *SyncCommand) Desc() string {
	return `Sync membership information`
}

func (c *SyncCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]

  Sync membership information from source and target system.

  Sync membership from google group to GitHub

  tlctl sync run \
	-src GoogleGroup \
	-dst GitHub \
	-group-mapping-config your-group-config.textproto \
	-user-mapping-config your-user-confif.textproto \
	-src-system-auth-token your-src-token \
	-dst-system-auth-token your-dst-token
`
}

func (c *SyncCommand) Flags() *cli.FlagSet {
	set := c.NewFlagSet()

	// Command options
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

	f.StringVar(&cli.StringVar{
		Name:    "src-system-auth-token",
		Target:  &c.sourceToken,
		Aliases: []string{"st", "src-token"},
		Usage:   `Token to authenticate with source system to read membership information`,
	})

	f.StringVar(&cli.StringVar{
		Name:    "dst-system-auth-token",
		Target:  &c.destinationToken,
		Aliases: []string{"dt", "dst-token"},
		Example: "user-mapping-config.textproto",
		Usage:   `Token to authenticate with destination system to write membership information`,
	})

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

	if ok := slices.Contains(allowedSourceSystem, strings.ToUpper(c.source)); !ok {
		return fmt.Errorf("source system %s not in allowed list: %s", c.source, strings.Join(allowedSourceSystem, ","))
	}

	if ok := slices.Contains(allowedDestinationSystem, strings.ToUpper(c.destination)); !ok {
		return fmt.Errorf("destination system %s not in allowed list: %s", c.destination, strings.Join(allowedDestinationSystem, ","))
	}

	if c.groupMappingConfig == "" {
		return fmt.Errorf("group mapping config file is not provided")
	}

	if c.userMappingConfig == "" {
		return fmt.Errorf("user mapping config file is not provided")
	}

	if c.sourceToken == "" {
		return fmt.Errorf("source system auth token is not provided")
	}

	if c.destinationToken == "" {
		return fmt.Errorf("destination system auth token is not provided")
	}

	// TODO(#72): create reader, writer base on cmd flags.
	// TODO(#71): create group and user mapping proto and textproto parser.

	return nil
}
