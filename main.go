package main

import (
	"os"
	"net"
	"github.com/zpatrick/go-config"
	"github.com/codegangsta/cli"
	"strings"
	"sort"
	"fmt"
)

func Run(ctx *cli.Context) {
	cfg := config.NewConfig([]config.Provider{})
	cfg.Providers = append(cfg.Providers, config.NewCLI(ctx, false))
	q := fmt.Sprintf("tasks.%s", ctx.Args().Get(0))
	hosts, _ := net.LookupHost(q)
	sort.Strings(hosts)
	if len(hosts) == 0 {
		fmt.Printf("Could not find IPs for '%s'\n", q)
		os.Exit(1)
	}
	out := ctx.String("out")
	switch out {
	case "bash":
		fmt.Println(strings.Join(hosts, " "))
	case "list":
		fmt.Println(strings.Join(hosts, ","))
	default:
		fmt.Printf("'%s' is not a valid output format. (bash)\n", out)
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "Golang library to help fishing information from moby"
	app.Usage = "go-fishermen [options]"
	app.Version = "0.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "out,o",
			Value: "bash",
			Usage: "Output format (only 'bash' or 'list' for now).",
		},
	}
	app.Action = Run
	app.Run(os.Args)
}
