package main

import (
	"fmt"
	"time"

	"github.com/gpmn/goembed"
)

// ExportTest :
func ExportTest(name string) string {
	fmt.Printf("Hello %s\n", name)
	return "good!"
}

// exports
var exports = map[string]interface{}{
	"ExportTest": ExportTest,
}

func main() {
	fmt.Printf(`1. input 'telnet 127.0.0.1 6666' in new console;
2. in telnet client, invoke 'ExportTest("beauty")';

   And, this program's stdout/stderr has been saved to /tmp/goembed.log;

   Wish this could help you to debug.
`)

	var ge goembed.GoEmbed
	ge.Serve("127.0.0.1:6666", "/tmp/goembed.log", "", exports)

	for {
		time.Sleep(10 * time.Second)
	}
}
