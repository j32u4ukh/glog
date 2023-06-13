package main

import (
	"fmt"
	"path"
	"strings"
)

func main() {
	str := "a/b/cd"
	result := path.Base(str)
	fmt.Println(result)
	hasSlash := strings.Contains(str, "/")
	fmt.Println(hasSlash) // Output: true
}
