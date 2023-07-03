package main

import (
	"flag"
	"github.com/elazarl/goproxy"
	"github.com/kballard/go-shellquote"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

type commands [][]string

func (cmds commands) String() string {
	var strs []string
	for _, cmd := range cmds {
		strs = append(strs, shellquote.Join(cmd...))
	}
	return strings.Join(strs, ", ")
}

func (cmds *commands) Set(value string) error {
	cmd, err := shellquote.Split(value)
	if err != nil {
		return err
	}

	*cmds = append(*cmds, cmd)
	return nil
}

func handleJukeboxControl(jukeboxSetCommands commands) func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	return func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		for _, jukeboxSetCommand := range jukeboxSetCommands {
			cmd := exec.Command(jukeboxSetCommand[0], jukeboxSetCommand[1:]...)
			cmd.Start()
		}

		return r, nil
	}
}

func isJukeboxControlSet(r *http.Request, ctx *goproxy.ProxyCtx) bool {
	return r.URL.Path == `/rest/jukeboxControl.view` && r.FormValue("action") == "set"
}

func main() {
	listenAddr := flag.String("listen-addr", "0.0.0.0:8080", "listen address")

	var jukeboxSetCommands commands
	flag.Var(&jukeboxSetCommands, "jukebox-set-command", "command to run when jukeboxControl 'set' is called (can be specified multiple times)")

	flag.Parse()

	proxy := goproxy.NewProxyHttpServer()

	proxy.OnRequest(goproxy.ReqConditionFunc(isJukeboxControlSet)).DoFunc(handleJukeboxControl(jukeboxSetCommands))

	log.Fatal(http.ListenAndServe(*listenAddr, proxy))
}
