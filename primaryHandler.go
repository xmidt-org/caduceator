package main

import (
	"os"
	"time"
)

type App struct {
}

func receiveEvents() {
	time.Sleep(5 * time.Second)
	os.Exit(0)
}

func receiveCutoff() {

}
