package video

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"time"
)

func TestCreateWindow(t *testing.T) {
	_, err := CreateWindow("Hello GDL", 100, 100, 400, 400, 0)
	assert.NoError(t, err)
	time.Sleep(5 * time.Second)
}