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

package credentials

import (
	"context"
	"crypto"
)

// KeyProvider provides a private key.
type KeyProvider interface {
	Key(ctx context.Context) ([]byte, error)
}

// SignerProvider provides a crypto.Signer.
type SignerProvider interface {
	Signer(ctx context.Context) (crypto.Signer, error)
}
