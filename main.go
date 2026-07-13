/*
Copyright © 2026 NAME HERE alessandro.dinato@gmail.com
*/
package main

import (
	"github.com/mtracer-project/mtracer/cmd"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cmd.Execute()
}
