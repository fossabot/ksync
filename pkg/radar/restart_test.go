package radar

import (
  "testing"
  "context"

  "github.com/stretchr/testify/assert"
  // "github.com/stretchr/testify/require"

)

func TestRestartMirror(t *testing.T) {
  radarserver := &radarServer{}
  cntx := context.Background()

  _, err := radarserver.RestartMirror(cntx, nil)

  assert.Error(t, err)
}
