package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	mod                     = windows.NewLazyDLL("user32.dll")
	procGetWindowText       = mod.NewProc("GetWindowTextW")
	procGetWindowTextLength = mod.NewProc("GetWindowTextLengthW")
	base                    string
	sources                 = make(map[string]int64)
)

type (
	// HANDLE does no idea what
	HANDLE uintptr
	// HWND same
	HWND HANDLE
)

func init() {
	base, _ = os.Getwd()
}

func main() {
	// parse db to sources
	parseDB()

	// if no arguments were passed
	if len(os.Args) < 2 {
		// handle KeyInterrupt
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			fmt.Println("saving data...")
			writeJSON("db.json", sources)
			os.Exit(1)
		}()

		// calculate spent time with hover app
		for {
			time.Sleep(1 * time.Second)
			if hwnd := getWindow("GetForegroundWindow"); hwnd != 0 {
				text := getWindowText(HWND(hwnd))
				if sources[text] != 0 {
					sources[text]++
				} else {
					sources[text] = 1
				}
			}
			fmt.Println(sources)
		}
		// if argument was passed
	} else {
		switch os.Args[1] {
		case "stats":
			fmt.Println(readToString("db.json"))
		}
	}
}

// parse JSON from db if exists,
// else - create new empty db
func parseDB() {
	if fileExists("db.json") {
		if data := readToString("db.json"); data != "" {
			if err := json.Unmarshal([]byte(data), &sources); err != nil {
				log.Fatalln(err)
			}
		}
	} else {
		f, err := os.Create("db.json")
		if err != nil {
			log.Fatalln(err)
		}
		// writeJSON("db.json", map[string]int64{"init": 0})
		defer f.Close()
	}
}

// check if file by specified path actually exists
func fileExists(f string) bool {
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		return err == nil
	}
	return false
}

// read file by specified path to single string
func readToString(f string) string {
	if b, err := ioutil.ReadFile(f); err == nil {
		return string(b)
	}
	return ""
}

// jsonify data and write to file
func writeJSON(f string, v map[string]int64) {
	b, err := json.Marshal(v)
	if err != nil {
		log.Fatalln(err)
	}
	err = ioutil.WriteFile(f, b, 0666)
	if err != nil {
		log.Fatalln(err)
	}
}

// have no idea what it actually does
func getWindowTextLength(hwnd HWND) int {
	ret, _, _ := procGetWindowTextLength.Call(
		uintptr(hwnd))

	return int(ret)
}

// get window label
func getWindowText(hwnd HWND) string {
	textLen := getWindowTextLength(hwnd) + 1

	buf := make([]uint16, textLen)
	procGetWindowText.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(textLen))

	text := syscall.UTF16ToString(buf)
	if strings.ContainsAny(text, "-") {
		full := strings.Split(text, "-")
		return full[len(full)-1]
	}
	return text
}

func getWindow(funcName string) uintptr {
	proc := mod.NewProc(funcName)
	hwnd, _, _ := proc.Call()
	return hwnd
}
