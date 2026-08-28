package main

import "regexp"

var (
	goExtMatch = regexp.MustCompile(`\.go$`)
	tsExtMatch = regexp.MustCompile(`\.tsx?$|\.jsx?$`)
)
