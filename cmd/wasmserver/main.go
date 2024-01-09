package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/cluster"
	"github.com/anthdm/hollywood/remote"
	"github.com/anthdm/raptor/internal/actrs"
	"github.com/anthdm/raptor/internal/config"
	"github.com/anthdm/raptor/internal/storage"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	var configFile string
	flagSet := flag.NewFlagSet("raptor", flag.ExitOnError)
	flagSet.StringVar(&configFile, "config", "config.toml", "")
	flagSet.Parse(os.Args[1:])

	err := config.Parse(configFile)
	if err != nil {
		log.Fatal(err)
	}

	var (
		user    = config.Get().Storage.User
		pw      = config.Get().Storage.Password
		dbname  = config.Get().Storage.Name
		host    = config.Get().Storage.Host
		port    = config.Get().Storage.Port
		sslmode = config.Get().Storage.SSLMode
	)
	store, err := storage.NewSQLStore(user, pw, dbname, host, port, sslmode)
	if err != nil {
		log.Fatal(err)
	}

	var (
		modCache    = storage.NewDefaultModCache()
		metricStore = store
	)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	remote := remote.New(config.Get().Cluster.WasmMemberAddr, nil)
	engine, err := actor.NewEngine(&actor.EngineConfig{
		Remote: remote,
	})
	if err != nil {
		log.Fatal(err)
	}

	cnf := cluster.NewConfig().
		WithRegion(config.Get().Cluster.Region).
		WithEngine(engine).
		WithID(config.Get().Cluster.ID).
		WithProvider(cluster.NewSelfManagedProvider(cluster.NewSelfManagedConfig())).
		WithRequestTimeout(time.Second * 2)

	c, err := cluster.New(cnf)
	c.RegisterKind(actrs.KindRuntime, actrs.NewRuntime(store, modCache), &cluster.KindConfig{})
	c.Engine().Spawn(actrs.NewMetric, actrs.KindMetric, actor.WithID("1"))
	c.Start()

	server := actrs.NewWasmServer(
		config.Get().WASMServerAddr,
		c,
		store,
		metricStore,
		modCache)
	c.Engine().Spawn(server, actrs.KindWasmServer)
	fmt.Printf("wasm server running\t%s\n", config.Get().WASMServerAddr)

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
	<-sigch
}
