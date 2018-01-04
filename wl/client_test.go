package wl

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Connect(t *testing.T) {
	c := &Client{}
	assert.NoError(t, c.Connect(""))
	for _, scr := range c.Screens {
		t.Log(scr)
	}
	w, err := c.CreateWindow()
	require.NotNil(t, w)
	assert.NoError(t, err)

	//t.Log(w)
	assert.NoError(t, w.SetTitle("Test Window"))
	w.Se
	assert.NoError(t, w.SetWindowGeometry(40, 40, 300, 300))
	assert.NoError(t, w.Commit())

	time.Sleep(4 * time.Second)
	assert.NoError(t, w.SetMaximized())

	time.Sleep(4 * time.Second)
	assert.NoError(t, w.SetMinimized())
	time.Sleep(1 * time.Second)
}
