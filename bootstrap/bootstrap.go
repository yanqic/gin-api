package bootstrap

import (
	"context"
	"flag"
	"log"
	"path"
	"path/filepath"
	"strings"

	app_config "github.com/why444216978/gin-api/config"
	"github.com/why444216978/gin-api/libraries/config"
	"github.com/why444216978/gin-api/libraries/etcd"
	"github.com/why444216978/gin-api/libraries/http"
	"github.com/why444216978/gin-api/libraries/jaeger"
	"github.com/why444216978/gin-api/libraries/logging"
	"github.com/why444216978/gin-api/libraries/orm"
	"github.com/why444216978/gin-api/libraries/redis"
	"github.com/why444216978/gin-api/libraries/registry"
	registry_etcd "github.com/why444216978/gin-api/libraries/registry/etcd"
	"github.com/why444216978/gin-api/resource"
)

var (
	conf = flag.String("conf", "conf_dev", "config path")
)

var confMap = map[string]struct{}{
	"conf_dev":      struct{}{},
	"conf_liantiao": struct{}{},
	"conf_qa":       struct{}{},
	"conf_online":   struct{}{},
}

func Bootstrap() {
	flag.Parse()

	initConfig()
	initApp()
	initLogger()
	initMysql("test_mysql")
	initRedis("default_redis")
	initJaeger()
	initEtcd()
	initServices()
	initHTTPRPC()
}

func initConfig() {
	confPath := *conf
	log.Println("The conf path is :" + confPath)

	if _, ok := confMap[confPath]; !ok {
		panic(confPath + " error")
	}

	var err error
	resource.Config = config.InitConfig(confPath, "toml")
	if err != nil {
		panic(err)
	}
}

func initApp() {
	if err := resource.Config.ReadConfig("app", "toml", &app_config.App); err != nil {
		panic(err)
	}
}

func initLogger() {
	var err error
	cfg := &logging.Config{}

	if err = resource.Config.ReadConfig("log/service", "toml", &cfg); err != nil {
		panic(err)
	}

	resource.ServiceLogger, err = logging.NewLogger(cfg)
	if err != nil {
		panic(err)
	}
}

func initMysql(db string) {
	var err error
	cfg := &orm.Config{}
	gormCfg := &logging.GormConfig{}

	if err = resource.Config.ReadConfig(db, "toml", cfg); err != nil {
		panic(err)
	}

	if err = resource.Config.ReadConfig("log/gorm", "toml", gormCfg); err != nil {
		panic(err)
	}

	gormLogger, err := logging.NewGorm(gormCfg)
	if err != nil {
		panic(err)
	}

	resource.TestDB, err = orm.NewOrm(cfg,
		orm.WithTrace(jaeger.GormTrace),
		orm.WithLogger(gormLogger),
	)
	if err != nil {
		panic(err)
	}
}

func initRedis(db string) {
	var err error
	cfg := &redis.Config{}
	logCfg := &logging.RedisConfig{}

	if err = resource.Config.ReadConfig(db, "toml", cfg); err != nil {
		panic(err)
	}
	if err = resource.Config.ReadConfig("log/redis", "toml", &logCfg); err != nil {
		panic(err)
	}

	logger, err := logging.NewRedisLogger(logCfg)
	if err != nil {
		panic(err)
	}

	rc := redis.NewClient(cfg)
	rc.AddHook(jaeger.NewJaegerHook())
	rc.AddHook(logger)
	resource.RedisCache = rc
}

func initJaeger() {
	var err error
	cfg := &jaeger.Config{}

	if err = resource.Config.ReadConfig("jaeger", "toml", cfg); err != nil {
		panic(err)
	}

	_, _, err = jaeger.NewJaegerTracer(cfg, app_config.App.AppName)
	if err != nil {
		panic(err)
	}
}

func initEtcd() {
	var err error
	cfg := &etcd.Config{}

	if err = resource.Config.ReadConfig("etcd", "toml", cfg); err != nil {
		panic(err)
	}

	resource.Etcd, err = etcd.NewClient(
		etcd.WithEndpoints(strings.Split(cfg.Endpoints, ";")),
		etcd.WithDialTimeout(cfg.DialTimeout),
	)
	if err != nil {
		panic(err)
	}
}

func initServices() {
	var (
		err   error
		dir   string
		files []string
	)

	if dir, err = filepath.Abs(*conf); err != nil {
		panic(err)
	}

	if files, err = filepath.Glob(filepath.Join(dir, "services", "*.toml")); err != nil {
		panic(err)
	}

	cfg := &registry.DiscoveryConfig{}
	discover := &registry_etcd.EtcdDiscovery{}
	for _, f := range files {
		f = path.Base(f)
		f = strings.TrimSuffix(f, path.Ext(f))

		if err = resource.Config.ReadConfig("services/"+f, "toml", cfg); err != nil {
			panic(err)
		}

		if discover, err = registry_etcd.NewDiscovery(
			registry_etcd.WithDiscoverClient(resource.Etcd.Client),
			registry_etcd.WithDiscoverConfig(cfg)); err != nil {
			panic(err)
		}

		err = discover.WatchService(context.Background())
		if err != nil {
			panic(err)
		}

		registry.Services[cfg.ServiceName] = discover
	}

	return
}

func initHTTPRPC() {
	var err error
	cfg := &logging.RPCConfig{}

	if err = resource.Config.ReadConfig("log/rpc", "toml", cfg); err != nil {
		panic(err)
	}

	rpcLogger, err := logging.NewRPCLogger(cfg)
	if err != nil {
		panic(err)
	}

	resource.HTTPRPC = http.New(http.WithLogger(rpcLogger))
	if err != nil {
		panic(err)
	}
}
