// Command daemon will run on the user's Mac and serve the private Web UI while
// bridging local agent terminals. The Web transport and terminal lifecycle are
// introduced by the first implementation milestone.
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
		fmt.Printf("code-remote-daemon %s\n", buildinfo.Version)
		return
	}

	fmt.Println("code-remote daemon scaffold: terminal transport is not implemented yet")
}
