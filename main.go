package main

import (
	"fmt"
	"sitemap/link"
	"strings"
)

var exampleHtml = `
<!DOCTYPE html>
<html>
<body>

<h2>The href Attribute</h2>

<p>HTML links are defined with the a tag. The link address is specified in the href attribute:</p>

<a href="https://github.com/RainbowHawk1993">Visit my github</a>

</body>
</html>
`

func main() {
	r := strings.NewReader(exampleHtml)
	links, err := link.Parse(r)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", links)
}
