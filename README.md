# router
baa module providers another router.

baa support replace router, like this:

```
package main

import (
	"github.com/go-baa/baa"
	"github.com/go-baa/router/regtree"
)

func main() {
	app := baa.Default()
	app.SetDI("router", regtree.New(app))

	app.Get("/view-:id(\\d+)", func(c *baa.Context) {
		c.String(200, c.Param("id"))
	})
	app.Get("/view-:id(\\d+)/project", func(c *baa.Context) {
		c.String(200, c.Param("id")+"/project")
	})
	app.Run(":1323")
}
```