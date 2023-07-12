package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/Dreamacro/clash/config"
	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/hub"
	"github.com/Dreamacro/clash/hub/executor"
	"github.com/Dreamacro/clash/log"

	"go.uber.org/automaxprocs/maxprocs"
)

var (
	flagset            map[string]bool // 参数是否被设置的标记 设置true 未设置false
	version            bool            // 是否显示系统版本号
	testConfig         bool            // 是否仅测试配置文件合法性
	homeDir            string          // 配置目录
	configFile         string          // 配置文件
	externalUI         string          // 在开启http服务后，设置是否启用ui接口
	externalController string          // 设置http服务监听地址和端口
	secret             string          // 设置http服务auth token
)

func init() {
	flag.StringVar(&homeDir, "d", "", "set configuration directory")
	flag.StringVar(&configFile, "f", "", "specify configuration file")
	flag.StringVar(&externalUI, "ext-ui", "", "override external ui directory")
	flag.StringVar(&externalController, "ext-ctl", "", "override external controller address")
	flag.StringVar(&secret, "secret", "", "override secret for RESTful API")
	flag.BoolVar(&version, "v", false, "show current version of clash")
	flag.BoolVar(&testConfig, "t", false, "test configuration and exit")
	flag.Parse()

	flagset = map[string]bool{}
	flag.Visit(func(f *flag.Flag) {
		flagset[f.Name] = true
	})
}

func main() {
	maxprocs.Set(maxprocs.Logger(func(string, ...any) {})) // 匹配容器cpu配额

	// 显示版本号，这里编译时，可以替换常量 Version BuildTime
	if version {
		fmt.Printf("Clash %s %s %s with %s %s\n", C.Version, runtime.GOOS, runtime.GOARCH, runtime.Version(), C.BuildTime)
		return
	}

	// 获取配置目录绝对路径
	if homeDir != "" {
		if !filepath.IsAbs(homeDir) {
			currentDir, _ := os.Getwd()
			homeDir = filepath.Join(currentDir, homeDir)
		}
		C.SetHomeDir(homeDir)
	}

	// 获取配置文件绝对路径
	if configFile != "" {
		if !filepath.IsAbs(configFile) {
			currentDir, _ := os.Getwd()
			configFile = filepath.Join(currentDir, configFile)
		}
		C.SetConfig(configFile)
	} else {
		confFile := filepath.Join(C.Path.HomeDir(), C.Path.Config())
		C.SetConfig(confFile)
	}

	// 初始化配置目录、配置文件、内存db
	if err := config.Init(C.Path.HomeDir()); err != nil {
		log.Fatalln("Initial configuration directory error: %s", err.Error())
	}

	// 验证配置文件合法性 yaml.Unmarshal
	if testConfig {
		if _, err := executor.Parse(); err != nil {
			log.Errorln(err.Error())
			fmt.Printf("configuration file %s test failed\n", C.Path.Config())
			os.Exit(1)
		}
		fmt.Printf("configuration file %s test is successful\n", C.Path.Config())
		return
	}

	// 判断是否启动http服务
	var options []hub.Option
	if flagset["ext-ui"] {
		options = append(options, hub.WithExternalUI(externalUI))
	}
	if flagset["ext-ctl"] {
		options = append(options, hub.WithExternalController(externalController))
	}
	if flagset["secret"] {
		options = append(options, hub.WithSecret(secret))
	}

	if err := hub.Parse(options...); err != nil {
		log.Fatalln("Parse config error: %s", err.Error())
	}

	// 监听信号，终止
	fmt.Println("start server!!!")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("end server!!!")
}
