package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"

	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
)

type window struct {
	Id    string
	Props string
	Tags  []string
}

func (win window) String() string {
	return win.Id + " " + win.Props + " " + strings.Join(win.Tags, "+")
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
	dirname := "/client"
	dirs, err := wm.readDir(dirname)
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
		fname = fmt.Sprintf("/client/%s/tags", dir.Name)
		tagstr, err := wm.readFile(fname)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wmiinav: read %s: %s", fname, err)
		}
		tags := []string{}
		for _, tag := range strings.Split(string(tagstr), "+") {
			if tag != "" {
				tags = append(tags, tag)
			}
		}

		wins = append(wins, window{Id: dir.Name, Props: string(props), Tags: tags})
	}

	return wins, nil
}

func (wm *wmii) SelectWindow(id string) error {
	return wm.writeFile("/tag/sel/ctl", []byte(fmt.Sprintf("select client %s\n", id)))
}

func (wm *wmii) View(tag string) error {
	return wm.writeFile("/ctl", []byte(fmt.Sprintf("view %s\n", tag)))
}

func (wm *wmii) CurrentTag() (string, error) {
	buf, err := wm.readFile("/ctl")
	if err != nil {
		return "", err
	}

	sc := bufio.NewScanner(bytes.NewReader(buf))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "view ") {
			return strings.TrimSpace(line[5:]), nil
		}
	}

	return "", sc.Err()
}

func (wm *wmii) AddTag(win *window, tag string) error {
	win.Tags = append(win.Tags, tag)
	sort.Strings(win.Tags)
	return wm.writeFile(fmt.Sprintf("/client/%s/tags", win.Id), []byte("+"+tag))
}

func (wm *wmii) readDir(name string) ([]*plan9.Dir, error) {
	fid, err := wm.fsys.Open(name, plan9.OREAD)
	if err != nil {
		return nil, err
	}
	defer fid.Close()
	return fid.Dirreadall()
}

func (wm *wmii) readFile(name string) ([]byte, error) {
	fid, err := wm.fsys.Open(name, plan9.OREAD)
	if err != nil {
		return nil, err
	}
	defer fid.Close()
	return ioutil.ReadAll(fid)
}

func (wm *wmii) writeFile(name string, data []byte) error {
	fid, err := wm.fsys.Open(name, plan9.OWRITE)
	if err != nil {
		return err
	}
	_, err = fid.Write(data)
	fid.Close()
	return err
}

func selectWindow(windows []window) (int, error) {
	items := make([]string, len(windows))
	for i := range items {
		items[i] = fmt.Sprintf("<%d> [%s] %s", i, strings.Join(windows[i].Tags, "+"), windows[i].Props)
	}

	return prompt(items)
}

func prompt(items []string) (int, error) {
	dmenu := exec.Command("dmenu", "-l", "7", "-i", "-b")

	in, err := dmenu.StdinPipe()
	if err != nil {
		return -1, err
	}

	go func() {
		for _, item := range items {
			fmt.Fprintln(in, item)
		}
		in.Close()
	}()

	out, err := dmenu.Output()
	if err != nil {
		return -1, err
	}

	if len(out) > 0 {
		sel := string(out[:len(out)-1])
		for i, item := range items {
			if item == sel {
				return i, nil
			}
		}
	}

	return -1, nil
}

func nav() {
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

	sel, err := selectWindow(windows)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if sel < 0 {
		return
	}

	win := windows[sel]
	ctag, _ := wm.CurrentTag()

	if len(win.Tags) == 0 {
		err := wm.AddTag(&win, ctag)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	ntag := win.Tags[0]
	for _, tag := range win.Tags {
		if tag == ctag {
			ntag = tag
		}
	}

	if ntag != ctag {
		if err := wm.View(ntag); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if err := wm.SelectWindow(win.Id); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	var cmd = ""
	if len(os.Args) < 2 {
		cmd = "nav"
	} else {
		cmd = os.Args[1]
	}

	switch cmd {
	case "nav":
		nav()
	default:
		fmt.Fprintf(os.Stderr, "wmiinav: unknown command %q\n", cmd)
		os.Exit(1)
	}
}
