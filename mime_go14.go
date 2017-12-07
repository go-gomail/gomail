// +build !go1.5

package gomail

import (
	"gopkg.in/alexcesaro/quotedprintable.v3"
)

// NOTE: this file is only used for < 1.5 ie ie 1.4--

var newQPWriter = quotedprintable.NewWriter

type mimeEncoder struct {
	quotedprintable.WordEncoder
}

var (
	bEncoding     = mimeEncoder{quotedprintable.BEncoding}
	qEncoding     = mimeEncoder{quotedprintable.QEncoding}
	lastIndexByte = func(s string, c byte) int {
		for i := len(s) - 1; i >= 0; i-- {

			if s[i] == c {
				return i
			}
		}
		return -1
	}
)
