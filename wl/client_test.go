package wl

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"encoding/json"
)

func TestClient_Connect(t *testing.T) {
	c := &Client{}
	assert.NoError(t, c.Connect(""))
	out, _ := json.MarshalIndent(c.globals, "", "  ")
	t.Log(string(out))
	}
