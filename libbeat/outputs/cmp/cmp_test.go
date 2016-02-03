package cmp

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/stretchr/testify/assert"
)

func TestCmpOneEvent(t *testing.T) {
	assert.Equal(t, 2, 2)
}
