// +build linux darwin

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
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
			log.Panic(err)
		}
		conf.File = filepath.Join(dir, ".config", "shadow", "config.json")
	}

	w := io.Writer(nil)
	if conf.Mode {
		w = os.Stdout
	}
	app, err := app.NewApp(conf.File, time.Minute, w)
	if err != nil {
		log.Panic(err)
	}

	if err := app.Run(); err != nil {
		log.Panic(err)
	}

	fmt.Println("shadow - a transparent proxy for Windows, Linux and macOS")
	fmt.Println("shadow is running...")

	systray.Run(func() {
		systray.SetIcon(icon)
		systray.SetTitle("")
		systray.SetTooltip("Shadow")

		items := make([]item, 0, 2)

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, os.Kill)
		items = append(items, item{
			condition: reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(sigCh),
			},
			function:  systray.Quit,
			terminate: true,
		})

		mQuit := systray.AddMenuItem("Exit", "Quit Shadow")
		items = append(items, item{
			condition: reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(mQuit.ClickedCh),
			},
			function:  systray.Quit,
			terminate: true,
		})

		go run(items)
	}, app.Close)

	fmt.Println("shadow is closing...")
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

type item struct {
	condition reflect.SelectCase
	function  func()
	terminate bool
}

func run(items []item) {
	cases := make([]reflect.SelectCase, len(items))
	for i, _ := range items {
		cases[i] = items[i].condition
	}
	for {
		i, _, _ := reflect.Select(cases)
		if item := items[i]; item.terminate {
			item.function()
			return
		} else {
			item.function()
		}
	}
}
