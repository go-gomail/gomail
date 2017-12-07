// +build go1.5

package gomail

// NOTE: this file is only used for 1.5+

import (
	"mime"
	"mime/quotedprintable"
	"strings"
)

var newQPWriter = quotedprintable.NewWriter

type mimeEncoder struct {
	mime.WordEncoder
}

var (
	bEncoding     = mimeEncoder{mime.BEncoding}
	qEncoding     = mimeEncoder{mime.QEncoding}
	lastIndexByte = strings.LastIndexByte
)
