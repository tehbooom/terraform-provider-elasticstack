// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package elasticsearch

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapConnectorUpdateConfigurationBody(t *testing.T) {
	t.Parallel()

	t.Run("wraps_field_map", func(t *testing.T) {
		t.Parallel()
		in := `{"buckets":{"value":"my-bucket","label":"List of buckets"}}`
		out, err := wrapConnectorUpdateConfigurationBody(in)
		require.NoError(t, err)
		var parsed struct {
			Configuration json.RawMessage `json:"configuration"`
		}
		require.NoError(t, json.Unmarshal([]byte(out), &parsed))
		require.JSONEq(t, in, string(parsed.Configuration))
	})

	t.Run("passes_through_configuration", func(t *testing.T) {
		t.Parallel()
		in := `{"configuration":{"buckets":{"value":"x"}}}`
		out, err := wrapConnectorUpdateConfigurationBody(in)
		require.NoError(t, err)
		require.Equal(t, in, out)
	})

	t.Run("passes_through_values", func(t *testing.T) {
		t.Parallel()
		in := `{"values":{"buckets":"my-bucket"}}`
		out, err := wrapConnectorUpdateConfigurationBody(in)
		require.NoError(t, err)
		require.Equal(t, in, out)
	})

	t.Run("rejects_empty", func(t *testing.T) {
		t.Parallel()
		_, err := wrapConnectorUpdateConfigurationBody("   ")
		require.Error(t, err)
	})

	t.Run("rejects_null", func(t *testing.T) {
		t.Parallel()
		_, err := wrapConnectorUpdateConfigurationBody("null")
		require.Error(t, err)
	})
}
