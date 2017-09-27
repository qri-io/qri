package main

import (
	"encoding/json"
	"fmt"
	"github.com/qri-io/qri/server"
	"log"
	"net/http"
	"os/exec"

	"github.com/fatih/color"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

//SocketEvent event
type SocketEvent struct {
	Event string
	Data  interface{}
}

//Backend Configuration returned by loadConfig
type configuration struct {
	name   string //Application Name
	server string
	port   string
}

//Globals
var done = make(chan bool, 1)      //Wait for Shutdown signal over websocket
var upgrader = websocket.Upgrader{ //Upgrader for websockets
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{"p0", "p1"},
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Loads configuration from file
// or inits values with default values
func loadConfig() configuration {
	viper.SetConfigName("config")

	// Paths to search for a config file
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		color.Set(color.FgRed)
		fmt.Println("No configuration file loaded - using defaults")
		color.Unset()
	}

	// default values
	viper.SetDefault("name", "")
	viper.SetDefault("server", "localhost")
	viper.SetDefault("port", "9109")

	// Write all params to stdout
	color.Set(color.FgGreen)
	fmt.Println("Loaded Configuration:")
	color.Unset()

	// Print config
	keys := viper.AllKeys()
	for i := range keys {
		key := keys[i]
		fmt.Println(key + ":" + viper.GetString(key))
	}
	fmt.Println("---")

	return configuration{
		name:   viper.GetString("name"),
		server: viper.GetString("server"),
		port:   viper.GetString("port")}
}

//Handles msgs to communicate with nodejs electron for rampup & shutdown
func socket(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	for {
		var event SocketEvent
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("ElectronSocket: [err]", err)
			break
		}

		//Handle Message
		err = json.Unmarshal(message, &event)
		if err != nil {
			log.Println("Unmashal: ", err)
			break
		}
		log.Printf("ElectronSocket: [received] %+v", event)

		//Shutdown Event
		if event.Event == "shutdown" {
			switch t := event.Data.(type) {
			case bool:
				if t {
					done <- true
				}
			}
		}
	}
}

func runServer() error {
	s, err := server.New(func(cfg *server.Config) {
		cfg.Port = "3000"
		cfg.MemOnly = false
		cfg.QriRepoPath = "/Users/b5/qri"
		cfg.FsStorePath = "/Users/b5/ipfs"
		// cfg.QriRepoPath = viper.GetString(QriRepoPath)
		// cfg.FsStorePath = viper.GetString(IpfsFsPath)
		cfg.Online = true
	})
	if err != nil {
		log.Println(err.Error())
		return err
	}

	return s.Serve()
}

func main() {
	config := loadConfig()

	var addr = config.server + ":" + config.port
	http.HandleFunc("/ui", socket)    //Endpoint for Electron startup/teardown
	go http.ListenAndServe(addr, nil) //Start websockets in goroutine

	log.Println("Starting qri...")
	go func() {
		err := runServer()
		log.Println(err.Error())
	}()

	log.Printf("Starting Electron...")
	cmd := exec.Command("electron", "app")
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	color.Set(color.FgGreen)
	log.Printf("%s succesfully started", config.name)
	color.Unset()

	<-done //Wait for shutdown signal
	color.Set(color.FgGreen)
	log.Printf("Shutting down...")
	color.Unset()
}
