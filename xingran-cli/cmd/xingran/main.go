// XingRan CLI 入口。
package main

import (
	"os"

	"github.com/xingran-next/ruoyi-go-backend/xingran-cli/cmd"
)

func main() {
	os.Exit(cmd.Execute(os.Args[1:]))
}
