package main

import (
	"errors"
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

type command []string

func (cmd command) String() string {
	return shellquote.Join(cmd...)
}

type commands []command

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

type rpc struct {
	path string
	cmd  command
}

func (rpc rpc) String() string {
	return rpc.path + " -> " + rpc.cmd.String()
}

type rpcs []rpc

func (rpcs rpcs) String() string {
	var strs []string
	for _, rpc := range rpcs {
		strs = append(strs, rpc.String())
	}
	return strings.Join(strs, ", ")
}

func (rpcs *rpcs) Set(value string) error {
	split := strings.SplitN(value, " ", 2)
	if len(split) != 2 {
		return errors.New("-add-rpc arguments should have the form \"path command\"")
	}

	path := split[0]
	cmd, err := shellquote.Split(split[1])
	if err != nil {
		return err
	}

	*rpcs = append(*rpcs, rpc{path: path, cmd: cmd})
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

func RpcRequestHandler(rpc rpc) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			go executeCommand(rpc.cmd)

			// 200 OK with no body.
		} else {
			http.Error(w, "Invalid request method.", 405)
		}
	}
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
	flag.Var(&jukeboxSetCommands, "jukebox-set-command", "command to run when jukeboxControl 'set' is called\n(can be specified multiple times)")

	var rpcs rpcs
	flag.Var(&rpcs, "add-rpc", "form: \"path command\", e.g. \"/rpc/volume-up /bin/volume.sh +10\".\nregisters a command to that will be run on a POST request to the given path.\n(can be specified multiple times)")

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

	for i := range rpcs {
		rpc := rpcs[i]
		http.HandleFunc(rpc.path, RpcRequestHandler(rpc))
	}

	log.Printf("Starting server on %q", *listenAddr)
	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}
