package main

import (
	"flag"
	"github.com/kballard/go-shellquote"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type commands []command
type command []string

func (cmd command) String() string {
	return shellquote.Join(cmd...)
}

func (cmds commands) String() string {
	var strs []string
	for _, cmd := range cmds {
		strs = append(strs, cmd.String())
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

func executeCommand(cmd command) {
	log.Printf("running %q", cmd.String())

	execCmd := exec.Command(cmd[0], cmd[1:]...)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Run()
}

func isJukeboxControlSet(r *http.Request) bool {
	return r.URL.Path == `/rest/jukeboxControl.view` && r.FormValue("action") == "set"
}

func ProxyRequestHandler(proxy *httputil.ReverseProxy, jukeboxSetCommands commands) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if isJukeboxControlSet(r) {
			for _, jukeboxSetCommand := range jukeboxSetCommands {
				go executeCommand(jukeboxSetCommand)
			}
		}

		proxy.ServeHTTP(w, r)
	}
}

func main() {
	subsonicAddr := flag.String("subsonic-addr", "", "address of subsonic server")
	listenAddr := flag.String("listen-addr", "0.0.0.0:8080", "listen address")

	var jukeboxSetCommands commands
	flag.Var(&jukeboxSetCommands, "jukebox-set-command", "command to run when jukeboxControl 'set' is called (can be specified multiple times)")

	flag.Parse()

	if *subsonicAddr == "" {
		log.Fatal("subsonic-addr must be provided")
	}

	subsonicUrl, err := url.Parse(*subsonicAddr)
	if err != nil {
		log.Fatalf("subsonic-addr %q is not a valid URL", *subsonicAddr)
	}

	proxy := httputil.NewSingleHostReverseProxy(subsonicUrl)

	http.HandleFunc("/", ProxyRequestHandler(proxy, jukeboxSetCommands))

	log.Printf("Starting server on %q", *listenAddr)
	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}
