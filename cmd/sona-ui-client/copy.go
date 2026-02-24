package main

import (
	"io"
	"os"
	"strings"
)

type Copy struct {
	Text string
	file *os.File
}

func (c *Copy) DataSourceSend(mimeType string, fd uintptr) {
	c.file = os.NewFile(fd, "clipboard")
	io.Copy(c.file, strings.NewReader(c.Text))
	c.file.Close()
}

func (c *Copy) DataSourceCancelled() {
	if c.file != nil {
		c.file.Close()
	}
}