package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"code.google.com/p/goplan9/plan9"
	"code.google.com/p/goplan9/plan9/client"
)

type window struct {
	Id    string
	Props string
}

func (win window) String() string {
	return win.Id + " " + win.Props
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
		wins = append(wins, window{Id: dir.Name, Props: string(props)})
	}

	return wins, nil
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

func selectWindow(windows []window) (int, error) {
	dmenu := exec.Command("dmenu", "-l",  "7")

	in, err := dmenu.StdinPipe()
	if err != nil {
		return -1, err
	}

	go func() {
		for _, win := range windows {
			fmt.Fprintln(in, win.String())
		}
		in.Close()
	}()

	out, err := dmenu.Output()
	if err != nil {
		return -1, err
	}

	if len(out) > 0 {
		sel := string(out[:len(out)-1])
		for i, win := range windows {
			if win.String() == sel {
				return i, nil
			}
		}
	}

	return -1, nil
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

	for _, win := range windows {
		fmt.Println(win)
	}

	sel, err := selectWindow(windows)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if sel >= 0 {
		fmt.Printf("select: %q\n", windows[sel])
	}

}
