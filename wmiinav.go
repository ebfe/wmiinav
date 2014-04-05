package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"code.google.com/p/goplan9/plan9"
	"code.google.com/p/goplan9/plan9/client"
)

func main() {
	c, err := client.MountService("wmii")
	if err != nil {
		panic(err)
	}

	fclient, err := c.Open("/client", plan9.OREAD)
	if err != nil {
		panic(err)
	}
	defer fclient.Close()

	dclients, err := fclient.Dirreadall()
	if err != nil {
		panic(err)
	}

	for _, dc := range dclients {
		if dc.Name == "sel" {
			continue
		}
		fid, err := c.Open(fmt.Sprintf("/client/%s/label", dc.Name), plan9.OREAD)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			continue
		}
		label, err := ioutil.ReadAll(fid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
		}
		fid.Close()
		fid, err = c.Open(fmt.Sprintf("/client/%s/props", dc.Name), plan9.OREAD)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
		}
		prop, err := ioutil.ReadAll(fid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
		}
		fid.Close()

		fmt.Printf("%s: %s %s\n", dc.Name, prop, label)
	}
}
