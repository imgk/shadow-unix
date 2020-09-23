// +build linux darwin

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/getlantern/systray"

	"github.com/imgk/shadow/app"
)

func main() {
	mode := flag.Bool("v", false, "enable verbose mode")
	file := flag.String("c", "", "config file")
	flag.Parse()

	if *file == "" {
		dir, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		*file = filepath.Join(dir, ".config", "shadow", "config.json")
	}

	app, err := app.NewApp(*file, time.Minute)
	if err != nil {
		panic(err)
	}
	if *mode {
		app.SetWriter(os.Stdout)
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
	case <-app.Done:
	}
}
