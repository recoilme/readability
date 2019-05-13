package main

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func TestRemove(t *testing.T) {
	f := readFile("html/3.htm")
	defer f.Close()
	n, err := html.Parse(f)
	assert.NoError(t, err)

	removeBad(n)
	ss := renderNode(n)
	log.Println(ss)
}
