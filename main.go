package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"strings"
	"fmt"
	"text/template"
)

// Server struct holds Droplet data for simpler syntax.
type Server struct {
	Hostname string
	IP       string
}

type Droplets struct {
	Droplets []Droplet
}

type Droplet struct {
	Name     string `json:"name"`
	Networks Network
}

type Network struct {
	Version []ProtocolStruct `json:"v4"`
}

type ProtocolStruct struct {
	IP string `json:"ip_address"`
}

const templ = `

#START do-go-ssh{{range .}}
Host {{.Hostname}}
	Hostname {{.IP}}
	Port 22
	User root
	IdentitiesOnly yes
{{end}}
#END do-go-ssh
`

// getServers connects to DO API and fetches droplet list.
func getServers(token string) []byte {

	req, err := http.NewRequest("GET", "https://api.digitalocean.com/v2/droplets", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if err != nil {
		panic(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return body

}

// parseServers unmarshals the JSON data from DO API and returns []Server struct
func parseServers(body []byte) []Server {

	var d Droplets
	err := json.Unmarshal(body, &d)
	if err != nil {
		panic(err)
	}

	var servers []Server
	for _, droplet := range d.Droplets {
		servers = append(servers, Server{droplet.Name, droplet.Networks.Version[0].IP})
	}

	return servers
}

// templateFrom creates a new text template from given []Server struct. It also fetches the old .ssh/config file 
// and parses it so that the old do-go-ssh values are removed. It then inserts the new values to the old file template 
// and finally returns the completed byte array.
func templateFrom(servers []Server) []byte {
	t, err := template.New("config").Parse(templ)
	if err != nil {
		panic(err)
	}

	var doc bytes.Buffer
	err = t.Execute(&doc, servers)
	if err != nil {
		panic(err)
	}

	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	file, err := ioutil.ReadFile(usr.HomeDir + "/.ssh/config")
	if err != nil {
		panic(err)
	}

	config := string(file)

	e := strings.Split(config, "#START do-go-ssh")
	h := strings.Split(config, "#END do-go-ssh")

	newconfig := e[0] + h[1]

	return []byte(strings.TrimSpace(newconfig) + doc.String())
}

// saveconfig saves the new .ssh/config file with file permissions 0612.
func saveconfig(document []byte) {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(usr.HomeDir+"/.ssh/config", document, 0612)
	if err != nil {
		panic(err)
	}
}

func main() {

	body := getServers(os.Args[1])

	servers := parseServers(body)

	document := templateFrom(servers)

	saveconfig(document)

	fmt.Println("SSH config file updated successfully.")
}