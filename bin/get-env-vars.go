package main

import (
	"fmt"
	"go_websocket_demo/pkg/websocket_api"

	"github.com/davecgh/go-spew/spew"
)

func main() {
	env := websocket_api.NewEnvironment()
	spew.Dump(env)

	fmt.Println(websocket_api.ResolveDBPassword(env))
}
