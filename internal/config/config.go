package config

import (
	"errors"
	"net"
	"os"

	"github.com/pelletier/go-toml/v2"
)

const defaultConfig = `
wasmServerAddr 		= "localhost:5000"
apiServerAddr 		= "localhost:3000"
storageDriver 		= "sqlite"
webuiHostAddr 		= "localhost:8080"
apiToken			= "foobarbaz"
authorization		= false

[cluster]
wasmMemberAddr 		= "localhost:6666"
id					= "wasm_member_1" 
region				= "eu-west"

[storage]
user 				= "postgres"
password 			= "postgres"
name 				= "postgres"
host				= "localhost"
port				= "5432"
sslmode 			= "disable"

`

func getDefaultConfig() Config {
	// parse default config
	var config Config
	err := toml.Unmarshal([]byte(defaultConfig), &config)
	if err != nil {
		panic(err)
	}
	return config
}

// Config holds the global configuration which is READONLY.
var config Config = getDefaultConfig()

type Storage struct {
	Name     string
	User     string
	Password string
	Host     string
	Port     string
	SSLMode  string
}

type Cluster struct {
	WasmMemberAddr string
	ID             string
	Region         string
}

type Config struct {
	APIServerAddr  string
	WASMServerAddr string
	StorageDriver  string
	WebUIHostAddr  string
	APIToken       string
	Authorization  bool

	Cluster Cluster
	Storage Storage
}

func Parse(path string) error {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile("config.toml", []byte(defaultConfig), os.ModePerm); err != nil {
			return err
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	err = toml.Unmarshal(b, &config)
	return err
}

func Get() Config {
	return config
}

// makeURL takes a host address and returns a http URL.
func makeURL(address string) string {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host = address
		port = ""
	}
	if host == "" {
		host = "0.0.0.0"
	}
	if port == "" || port == "http" {
		port = "80"
	}
	return "http://" + net.JoinHostPort(host, port)
}

func GetWasmUrl() string {
	return makeURL(config.WASMServerAddr)
}

func GetApiUrl() string {
	return makeURL(config.APIServerAddr)
}

func GetWebUIUrl() string {
	return makeURL(config.WebUIHostAddr)
}
