package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSemver(t *testing.T) {
	ver := semver("6aa57cbe96b859c5d3d9e8ddd0a16b1e248cb7a2",
		time.Date(2019, time.April, 9, 10, 40, 7, 100, time.UTC),
		"dirtyworktree")
	assert.Regexp(t, "0.0.0-20190409104007-6aa57cbe96b8-dirty-dirtyworktree", ver)

	ver2 := semver(
		"000000000000",
		time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC),
		"")
	assert.Equal(t, "0.0.0-19700101000000-000000000000", ver2)

	ver3 := semver("6aa57cbe96b859c5d3d9e8ddd0a16b1e248cb7a2",
		time.Date(2019, time.April, 9, 10, 40, 7, 100, time.UTC),
		"")
	assert.Equal(t, "0.0.0-20190409104007-6aa57cbe96b8", ver3)
}
