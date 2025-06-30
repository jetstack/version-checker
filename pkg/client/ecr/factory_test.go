package ecr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHost(t *testing.T) {
	tests := map[string]struct {
		host  string
		expIs bool
	}{
		"an empty host should be false": {
			host:  "",
			expIs: false,
		},
		"random string should be false": {
			host:  "foobar",
			expIs: false,
		},
		"random string with dots should be false": {
			host:  "foobar.foo",
			expIs: false,
		},
		"just amazonawsaws.com should be false": {
			host:  "amazonaws.com",
			expIs: false,
		},
		"ecr.foo.amazonaws.com with random sub domains should be false": {
			host:  "bar.ecr.foo.amazonaws.com",
			expIs: false,
		},
		"dkr.ecr.foo.amazonaws.com with random sub domains should be false": {
			host:  "dkr.ecr.foo.amazonaws.com",
			expIs: false,
		},
		"hello123.dkr.ecr.foo.amazonaws.com true": {
			host:  "hello123.dkr.ecr.foo.amazonaws.com",
			expIs: true,
		},
		"123hello.dkr.ecr.foo.amazonaws.com true": {
			host:  "123hello.dkr.ecr.foo.amazonaws.com",
			expIs: true,
		},
		"hello123.dkr.ecr.foo.amazonaws.com.cn true": {
			host:  "hello123.dkr.ecr.foo.amazonaws.com.cn",
			expIs: true,
		},
		"123hello.dkr.ecr.foo.amazonaws.com.cn true": {
			host:  "123hello.dkr.ecr.foo.amazonaws.com.cn",
			expIs: true,
		},
		"123hello.hello.dkr.ecr.foo.amazonaws.com false": {
			host:  "123hello.hello.dkr.ecr.foo.amazonaws.com",
			expIs: false,
		},
		"123hello.dkr.ecr.foo.amazonaws.comfoo false": {
			host:  "123hello.dkr.ecr.foo.amazonaws.comfoo",
			expIs: false,
		},
	}

	handler := new(Factory)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expIs, handler.IsHost(test.host))
		})
	}
}
