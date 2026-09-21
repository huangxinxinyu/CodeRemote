// Command relay is a legacy scaffold from the superseded public-relay design.
// It is not part of the Web + Tailscale MVP.
package main

import (
	"flag"
	"fmt"

	"github.com/huangxinxinyu/CodeRemote/internal/buildinfo"
)

func main() {
	showVersion := flag.Bool("version", false, "print version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("code-remote-relay %s\n", buildinfo.Version)
		return
	}

	fmt.Println("code-remote relay scaffold: connection routing is not implemented yet")
}
