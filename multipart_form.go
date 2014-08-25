// Copyright (c) 2014 Brian Nelson. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

type MultipartForm struct {
	Fields map[string]string
	Files  map[string]string
}

func NewMultipartForm() *MultipartForm {
	return &MultipartForm{Files: make(map[string]string), Fields: make(map[string]string)}
}
