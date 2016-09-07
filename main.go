package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/iris-contrib/template/django"
	"github.com/kataras/iris"
	"github.com/spf13/viper"
)

var sokets []iris.WebsocketConnection

func main() {
	viper.SetConfigType("yaml")
	viper.SetConfigName("config") // name of config file (without extension)
	viper.AddConfigPath(".")
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	fmt.Println(viper.Get("commands"))

	iris.Static("/js", "./static/js", 1)
	iris.UseTemplate(django.New()).Directory("./templates", ".html")
	iris.Get("/", index)
	iris.Get("/run/:index", run)
	iris.Config.Websocket.Endpoint = "/ws"
	iris.Websocket.OnConnection(func(c iris.WebsocketConnection) {
		sokets = append(sokets, c)
		c.EmitMessage([]byte("connected"))
	})
	iris.Listen("0.0.0.0:" + viper.GetString("port"))
}

func index(ctx *iris.Context) {
	ctx.Render("index.html", map[string]interface{}{"commands": viper.Get("commands"), "host": ctx.HostString()})
}

func run(ctx *iris.Context) {
	n, _ := ctx.ParamInt("index")
	command := viper.Get("commands").([]interface{})[n]
	fmt.Println(n, command)
	cmdLine := command.(map[interface{}]interface{})["cmd"].(string)
	fmt.Println(cmdLine)
	tokens := strings.Split(cmdLine, " ")
	cmd := exec.Command(tokens[0], tokens[1:]...)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating StdoutPipe for Cmd", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			t := scanner.Text()
			fmt.Printf(">>> | %s\n", t)
			t = t + "\n"
			sokets[0].To(iris.All).Emit("out", []byte(t))
		}
	}()

	sokets[0].To(iris.All).Emit("out", []byte(">>> "+cmdLine))
	err = cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
		os.Exit(1)
	}

	err = cmd.Wait()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error waiting for Cmd", err)
		os.Exit(1)
	}
	ctx.Write(cmdLine)
}
