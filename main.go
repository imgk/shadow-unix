// +build linux darwin

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/getlantern/systray"

	//protocols
	_ "github.com/imgk/shadow/protocol/http"
	_ "github.com/imgk/shadow/protocol/shadowsocks"
	_ "github.com/imgk/shadow/protocol/socks"
	_ "github.com/imgk/shadow/protocol/trojan"

	"github.com/imgk/shadow/app"
)

func main() {
	var conf struct {
		Mode bool
		File string
	}
	flag.BoolVar(&conf.Mode, "v", false, "enable verbose mode")
	flag.StringVar(&conf.File, "c", "", "config file")
	flag.Parse()

	if conf.File == "" {
		dir, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		conf.File = filepath.Join(dir, ".config", "shadow", "config.json")
	}

	w := io.Writer(nil)
	if conf.Mode {
		w = os.Stdout
	}
	app, err := app.NewApp(conf.File, time.Minute, w)
	if err != nil {
		panic(err)
	}

	if err := app.Run(); err != nil {
		panic(err)
	}

	fmt.Println("shadow - a transparent proxy for Windows, Linux and macOS")
	fmt.Println("shadow is running...")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, os.Kill)

	systray.Run(func() {
		systray.SetIcon(icon)
		systray.SetTitle("")
		systray.SetTooltip("Shadow")

		mQuit := systray.AddMenuItem("Exit", "Quit Shadow")
		go func () {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, os.Kill)
			for {
				select {
				case <-mQuit.ClickedCh:
					systray.Quit()
					return
				case <-sigCh:
					systray.Quit()
					return
				}
			}
		}()
	}, func(){
		fmt.Println("shadow is closing...")
		app.Close()
	})

	select {
	case <-time.After(time.Second * 10):
		buf := make([]byte, 1024)
		for {
			n := runtime.Stack(buf, true)
			if n < len(buf) {
				buf = buf[:n]
				break
			}
			buf = make([]byte, 2*len(buf))
		}
		lines := bytes.Split(buf, []byte{'\n'})
		fmt.Println("Failed to shutdown after 10 seconds. Probably dead locked. Printing stack and killing.")
		for _, line := range lines {
			if len(bytes.TrimSpace(line)) > 0 {
				fmt.Println(string(line))
			}
		}
		os.Exit(777)
	case <-app.Done():
	}
}
