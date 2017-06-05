package main

import (
	"github.com/reechou/share-liebian/config"
	"github.com/reechou/share-liebian/controller"
)

func main() {
	controller.NewLogic(config.NewConfig()).Run()
}
