// XingRan CLI 入口。
package main

import (
	"os"

	"github.com/xingran-next/xingran-go-backend/xingran-cli/cmd"
)

func main() {
	os.Exit(cmd.Execute(os.Args[1:]))
}
