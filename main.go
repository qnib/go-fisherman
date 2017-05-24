package main

import (
	"os"
	"net"
	"github.com/zpatrick/go-config"
	"github.com/codegangsta/cli"
	"strings"
	"sort"
	"fmt"
	"errors"
	"time"
	"log"
	"text/template"
	"bytes"
)

var (
	fishermanTemplates = map[string]string{
		"etcd": "{{.ServiceName}}.{{.TaskSlot}}.{{.TaskID}}=http://{{.IP}}:2380",
	}
)

type TaskInfo struct {
	ServiceName	string
	TaskSlot	string
	TaskID		string
	IP			string
	NetworkName	string
	HostName	string
}

func (t *TaskInfo) String() string {
	res := []string{}
	res = append(res, fmt.Sprintf("%-15s : %s", "ServiceName", t.ServiceName))
	res = append(res, fmt.Sprintf("%-15s : %s", "TaskSlot", t.TaskSlot))
	res = append(res, fmt.Sprintf("%-15s : %s", "TaskID", t.TaskID))
	return strings.Join(res, "\n")
}

type Fisherman struct {
	Ctx *cli.Context
}

func NewFisherman(ctx *cli.Context) Fisherman {
	return Fisherman{
		Ctx: ctx,
	}
}

func levelToInt(level string) int {
	switch level {
	case "error":
		return 3
	case "warn":
		return 4
	case "notice":
		return 5
	case "info":
		return 6
	case "debug":
		return 7
	default:
		fmt.Printf("Can not resolve '%s' to level integer\n", level)
		os.Exit(1)
	}
	return 0
}
func (f *Fisherman) Log(level, msg string) {
	lint := levelToInt(level)
	logLevel := levelToInt(f.Ctx.String("log-level"))
	if logLevel >= lint {
		log.Printf("[%-5s] %s", level, msg)
	}
}

func (f *Fisherman) resolveTask(ip string) (ti TaskInfo, err error) {
	addrs, _ := net.LookupAddr(ip)
	if len(addrs) == 0 {
		return ti, errors.New(fmt.Sprintf("Could not identify exctly on task for '%s': got %v", ip, addrs))
	}
	slice := strings.Split(addrs[0], ".")
	ti.IP = ip
	ti.ServiceName = slice[0]
	ti.TaskSlot = slice[1]
	ti.TaskID = slice[2]
	ti.NetworkName = slice[3]
	ti.HostName = fmt.Sprintf("%s.%s.%s", ti.ServiceName, ti.TaskSlot, ti.TaskID)
	f.Log("debug", fmt.Sprintf("Resolved '%s' to '%s'", ip, ti))
	return
}

func (f *Fisherman) createHealthCheckOverwrite() {
	hdir := f.Ctx.String("healthcheck-dir")
	fpath := fmt.Sprintf("%s/force_true", hdir)
	if _, err := os.Stat(fpath); err == nil {
		return
	}
	w, err := os.Create(fpath)
	if err != nil {
		panic(err)
	}
	defer w.Close()

	f.Log("debug", fmt.Sprintf("File '%s' created!", fpath))
}

func (f *Fisherman) fetchIPs() []string {
	mintasks := f.Ctx.Int("mintasks")
	delay := time.Duration(f.Ctx.Int("delay"))*time.Second
	srv := f.Ctx.Args().Get(0)
	ips := []string{}
	for {
		q := fmt.Sprintf("tasks.%s", srv)
		ips, _ = net.LookupHost(q)
		sort.Strings(ips)
		if mintasks == 0 {
			if len(ips) == 0 {
				fmt.Printf("Could not find IPs for '%s'\n", q)
				os.Exit(1)
			}
			break

		} else if len(ips) < mintasks {
			f.Log("debug", fmt.Sprintf("Only found %d ips so far %v, expect %d", len(ips), ips, mintasks))
			f.createHealthCheckOverwrite()
			time.Sleep(delay)
			continue
		} else {
			break
		}
	}
	return ips
}

func (f *Fisherman) Run() {
	srv := f.Ctx.Args().Get(0)
	if srv == "" {
		f.Log("error", "please provide service name to look for as an argument.")
		os.Exit(1)
	}
	res := []string{}
	printTaskIP := f.Ctx.Bool("print-task-ip")
	out := f.Ctx.String("out")
	tmpl := f.Ctx.String("template")
	if val, ok := fishermanTemplates[tmpl]; ok {
		f.Log("debug", fmt.Sprintf("evaluated '%s' to '%s'", tmpl, val))
		tmpl = val

	}
	ips := f.fetchIPs()
	hostname, _ := os.Hostname()
	for _, ip := range ips {
		task, err := f.resolveTask(ip)
		if printTaskIP && hostname == task.HostName {
			fmt.Println(task.IP)
			os.Exit(0)
		}
		if err != nil {
			f.Log("error", err.Error())
			test, _ := net.LookupAddr(ip)
			f.Log("debug", strings.Join(test, " "))
		}
		f.Log("debug", fmt.Sprintf("Running Template: %s", tmpl))
		t := template.Must(template.New("tmpl").Parse(tmpl))
		buf := new(bytes.Buffer)
		err = t.Execute(buf, task)
		if err != nil {
			f.Log("error", fmt.Sprintf("Error during rendering '%s': %s", tmpl, err.Error()))
		}
		res = append(res, buf.String())
	}
	switch out {
	case "bash":
		fmt.Println(strings.Join(res, " "))
	case "list":
		fmt.Println(strings.Join(res, ","))
	default:
		f.Log("error", fmt.Sprintf("'%s' is not a valid output format. (bash)", out))
	}

}

func Run(ctx *cli.Context) {
	cfg := config.NewConfig([]config.Provider{})
	cfg.Providers = append(cfg.Providers, config.NewCLI(ctx, false))
	f := NewFisherman(ctx)
	f.Run()
}

func main() {
	app := cli.NewApp()
	app.Name = "Golang library to help fishing information from moby"
	app.Usage = "go-fishermen [options] <service>"
	app.Version = "0.0.2"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "out,o",
			Value: "bash",
			Usage: "Output format (only 'bash' or 'list' for now).",
			EnvVar: "FISHERMAN_OUT",
		},
		cli.StringFlag{
			Name:  "template,t",
			Value: "{{.IP}}",
			Usage: "Golang Template. (e.g. '{{.ServiceName}}.{{.TaskSlot}}.{{.TaskID}}={{.IP}}')",
			EnvVar: "FISHERMAN_TEMPLATE",
		},
		cli.IntFlag{
			Name:  "mintasks",
			Value: 0,
			Usage: "Expected amount of task to discover. While in this mode the local healhcheck will be set to TRUE (0: disable)",
			EnvVar: "FISHERMAN_MIN_TASKS",
		},
		cli.IntFlag{
			Name:  "delay",
			Value: 1,
			Usage: "Delay (seconds) between lookups",
			EnvVar: "FISHERMAN_DELAY",
		},
		cli.StringFlag{
			Name:   "healthcheck-dir",
			Value:  "/opt/healthcheck/",
			Usage:  "Healhcheck directory in which ./force_true overwrites the healthcheck to become true",
			EnvVar: "HEALTHCHECK_DIR",
		},
		cli.BoolFlag{
			Name:   "healthcheck-overwrite",
			Usage:  "If set the healthcheck can be overwriten if mintasks is not 0",
			EnvVar: "ALLOW_HEALTHCHECK_OVERWRITE",
		},
		cli.BoolFlag{
			Name:   "print-task-ip",
			Usage:  "Prints the IP of the task matching the hostname",
		},
		cli.StringFlag{
			Name:   "log-level",
			Value:  "warn",
			Usage:  "Log level (warn: silent)",
			EnvVar: "LOG_LEVEL",
		},
	}
	app.Action = Run
	app.Run(os.Args)
}
