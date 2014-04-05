package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"code.google.com/p/goplan9/plan9"
	"code.google.com/p/goplan9/plan9/client"
)

type window struct {
	Id    string
	Props string
}

type wmii struct {
	conn *client.Conn
	fsys *client.Fsys
}

func newWmii() (*wmii, error) {
	conn, err := client.DialService("wmii")
	if err != nil {
		return nil, err
	}

	fsys, err := conn.Attach(nil, "", "")
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &wmii{conn: conn, fsys: fsys}, nil
}

func (wm *wmii) Close() error {
	return wm.conn.Close()
}

func (wm *wmii) Windows() ([]window, error) {
	fid, err := wm.fsys.Open("/client", plan9.OREAD)
	if err != nil {
		return nil, err
	}
	defer fid.Close()

	dirs, err := fid.Dirreadall()
	if err != nil {
		return nil, err
	}

	wins := make([]window, 0, len(dirs))
	for _, dir := range dirs {
		if dir.Name == "sel" {
			continue
		}
		fname := fmt.Sprintf("/client/%s/props", dir.Name)
		props, err := wm.readFile(fname)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wmiinav: read %s: %s", fname, err)
		}
		fid.Close()
		wins = append(wins, window{Id: dir.Name, Props: string(props)})
	}

	return wins, nil
}

func (wm *wmii) readFile(name string) ([]byte, error) {
	fid, err := wm.fsys.Open(name, plan9.OREAD)
	if err != nil {
		return nil, err
	}
	defer fid.Close()
	return ioutil.ReadAll(fid)
}

func main() {
	wm, err := newWmii()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer wm.Close()

	windows, err := wm.Windows()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for i, win := range windows {
		fmt.Printf("%02x %s\n", i, win.Props)
	}
}
