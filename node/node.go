package node

import (
	"errors"
	"fmt"
	"github.com/QinPengLin/repro-origin/cluster"
	"github.com/QinPengLin/repro-origin/console"
	"github.com/QinPengLin/repro-origin/log"
	"github.com/QinPengLin/repro-origin/profiler"
	"github.com/QinPengLin/repro-origin/service"
	"github.com/QinPengLin/repro-origin/util/buildtime"
	"github.com/QinPengLin/repro-origin/util/sysprocess"
	"github.com/QinPengLin/repro-origin/util/timer"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var sig chan os.Signal
var nodeId string
var preSetupService []service.IService //预安装
var preSetupTemplateService []func() service.IService
var profilerInterval time.Duration
var configDir = "./config/"
var NodeIsRun = false

const (
	SingleStop   syscall.Signal = 10
	SignalRetire syscall.Signal = 12
)

type BuildOSType = int8

const (
	Windows BuildOSType = 0
	Linux   BuildOSType = 1
	Mac     BuildOSType = 2
)

func init() {
	sig = make(chan os.Signal, 4)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, SingleStop, SignalRetire)

	console.RegisterCommandBool("help", false, "<-help> This help.", usage)
	console.RegisterCommandString("name", "", "<-name nodeName> Node's name.", setName)
	console.RegisterCommandString("start", "", "<-start nodeid=nodeid> Run originserver.", startNode)
	console.RegisterCommandString("stop", "", "<-stop nodeid=nodeid> Stop originserver process.", stopNode)
	console.RegisterCommandString("retire", "", "<-retire nodeid=nodeid> retire originserver process.", retireNode)
	console.RegisterCommandString("config", "", "<-config path> Configuration file path.", setConfigPath)
	console.RegisterCommandString("pprof", "", "<-pprof ip:port> Open performance analysis.", setPprof)
}

func notifyAllServiceRetire() {
	service.NotifyAllServiceRetire()
}

func usage(val interface{}) error {
	ret := val.(bool)
	if ret == false {
		return nil
	}

	if len(buildtime.GetBuildDateTime()) > 0 {
		fmt.Fprintf(os.Stderr, "Welcome to Origin(build info: %s)\nUsage: originserver [-help] [-start node=1] [-stop] [-config path] [-pprof 0.0.0.0:6060]...\n", buildtime.GetBuildDateTime())
	} else {
		fmt.Fprintf(os.Stderr, "Welcome to Origin\nUsage: originserver [-help] [-start node=1] [-stop] [-config path] [-pprof 0.0.0.0:6060]...\n")
	}

	console.PrintDefaults()
	return nil
}

func setName(_ interface{}) error {
	return nil
}

func setPprof(val interface{}) error {
	listenAddr := val.(string)
	if listenAddr == "" {
		return nil
	}

	go func() {
		err := http.ListenAndServe(listenAddr, nil)
		if err != nil {
			panic(fmt.Errorf("%+v", err))
		}
	}()

	return nil
}

func setConfigPath(val interface{}) error {
	configPath := val.(string)
	if configPath == "" {
		return nil
	}
	_, err := os.Stat(configPath)
	if err != nil {
		return fmt.Errorf("cannot find file path %s", configPath)
	}

	cluster.SetConfigDir(configPath)
	configDir = configPath
	return nil
}

func getRunProcessPid(nodeId string) (int, error) {
	f, err := os.OpenFile(fmt.Sprintf("%s_%s.pid", os.Args[0], nodeId), os.O_RDONLY, 0600)
	defer f.Close()
	if err != nil {
		return 0, err
	}

	pidByte, errs := io.ReadAll(f)
	if errs != nil {
		return 0, errs
	}

	return strconv.Atoi(string(pidByte))
}

func writeProcessPid(nodeId string) {
	//pid
	f, err := os.OpenFile(fmt.Sprintf("%s_%s.pid", os.Args[0], nodeId), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	defer f.Close()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	} else {
		_, err = f.Write([]byte(fmt.Sprintf("%d", os.Getpid())))
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(-1)
		}
	}
}

func GetNodeId() string {
	return nodeId
}

func initNode(id string) {
	//1.初始化集群
	nodeId = id
	err := cluster.GetCluster().Init(GetNodeId(), Setup)
	if err != nil {
		log.Error("Init cluster fail", log.ErrorAttr("error", err))
		os.Exit(1)
	}

	err = initLog()
	if err != nil {
		return
	}

	//2.顺序安装服务
	serviceOrder := cluster.GetCluster().GetLocalNodeInfo().ServiceList
	for _, serviceName := range serviceOrder {
		bSetup := false

		//判断是否有配置模板服务
		splitServiceName := strings.Split(serviceName, ":")
		if len(splitServiceName) == 2 {
			serviceName = splitServiceName[0]
			templateServiceName := splitServiceName[1]
			for _, newSer := range preSetupTemplateService {
				ser := newSer()
				ser.OnSetup(ser)
				if ser.GetName() == templateServiceName {
					ser.SetName(serviceName)
					ser.Init(ser, cluster.GetRpcClient, cluster.GetRpcServer, cluster.GetCluster().GetServiceCfg(ser.GetName()))
					service.Setup(ser)

					bSetup = true
					break
				}
			}

			if bSetup == false {
				log.Error("Template service not found", log.String("service name", serviceName), log.String("template service name", templateServiceName))
				os.Exit(1)
			}
		}

		for _, s := range preSetupService {
			if s.GetName() != serviceName {
				continue
			}
			bSetup = true
			pServiceCfg := cluster.GetCluster().GetServiceCfg(s.GetName())
			s.Init(s, cluster.GetRpcClient, cluster.GetRpcServer, pServiceCfg)

			service.Setup(s)
		}

		if bSetup == false {
			log.Fatal("Service name " + serviceName + " configuration error")
		}
	}

	//3.service初始化
	service.Init()
}

func initLog() error {
	localNodeInfo := cluster.GetCluster().GetLocalNodeInfo()
	//设置日志文件的路径
	if err := setLogPath(localNodeInfo.LogCfg.Path); err != nil {
		return err
	}
	//设置日志级别
	if err := setLevel(localNodeInfo.LogCfg.Level); err != nil {
		return err
	}
	//设置日志输出格式
	if err := setLogEncoder(localNodeInfo.LogCfg.Encoder); err != nil {
		return err
	}
	//设置日志输出方式
	if err := setLogOutputType(localNodeInfo.LogCfg.OutputType); err != nil {
		return err
	}
	//设置文件最大日志大小
	if err := setLogSize(localNodeInfo.LogCfg.FlieSize); err != nil {
		return err
	}

	logger, err := log.NewLogger(localNodeInfo.NodeId)
	if err != nil {
		fmt.Printf("cannot create log file!\n")
		return err
	}
	log.SetLogger(logger)
	return nil
}

func Start() {
	err := console.Run(os.Args)
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}
}

func retireNode(args interface{}) error {
	//1.解析参数
	param := args.(string)
	if param == "" {
		return nil
	}

	sParam := strings.Split(param, "=")
	if len(sParam) != 2 {
		return fmt.Errorf("invalid option %s", param)
	}
	if sParam[0] != "nodeid" {
		return fmt.Errorf("invalid option %s", param)
	}
	nId := strings.TrimSpace(sParam[1])
	if nId == "" {
		return fmt.Errorf("invalid option %s", param)
	}

	processId, err := getRunProcessPid(nId)
	if err != nil {
		return err
	}

	RetireProcess(processId)
	return nil
}

func stopNode(args interface{}) error {
	//1.解析参数
	param := args.(string)
	if param == "" {
		return nil
	}

	sParam := strings.Split(param, "=")
	if len(sParam) != 2 {
		return fmt.Errorf("invalid option %s", param)
	}
	if sParam[0] != "nodeid" {
		return fmt.Errorf("invalid option %s", param)
	}
	nId := strings.TrimSpace(sParam[1])
	if nId == "" {
		return fmt.Errorf("invalid option %s", param)
	}

	processId, err := getRunProcessPid(nId)
	if err != nil {
		return err
	}

	KillProcess(processId)
	return nil
}

func startNode(args interface{}) error {
	//1.解析参数
	param := args.(string)
	if param == "" {
		return nil
	}

	sParam := strings.Split(param, "=")
	if len(sParam) != 2 {
		return fmt.Errorf("invalid option %s", param)
	}
	if sParam[0] != "nodeid" {
		return fmt.Errorf("invalid option %s", param)
	}
	strNodeId := strings.TrimSpace(sParam[1])
	if strNodeId == "" {
		return fmt.Errorf("invalid option %s", param)
	}

	for {
		processId, pErr := getRunProcessPid(strNodeId)
		if pErr != nil {
			break
		}

		name, cErr := sysprocess.GetProcessNameByPID(int32(processId))
		myName, mErr := sysprocess.GetMyProcessName()
		//当前进程名获取失败，不应该发生
		if mErr != nil {
			log.Info("get my process's name is error", log.ErrorAttr("err", mErr))
			os.Exit(-1)
		}

		//进程id存在，而且进程名也相同，被认为是当前进程重复运行
		if cErr == nil && name == myName {
			log.Info("repeat runs are not allowed", log.String("nodeId", strNodeId), log.Int("processId", processId))
			os.Exit(-1)
		}
		break
	}

	//2.记录进程id号
	writeProcessPid(strNodeId)
	timer.StartTimer(10*time.Millisecond, 1000000)

	//3.初始化node
	initNode(strNodeId)
	log.Info("Start running server.")

	//4.运行service
	service.Start()

	//5.运行集群
	cluster.GetCluster().Start()

	//6.监听程序退出信号&性能报告

	var pProfilerTicker *time.Ticker = &time.Ticker{}
	if profilerInterval > 0 {
		pProfilerTicker = time.NewTicker(profilerInterval)
	}

	NodeIsRun = true
	for NodeIsRun {
		select {
		case s := <-sig:
			signal := s.(syscall.Signal)
			if signal == SignalRetire {
				log.Info("receipt retire signal.")
				notifyAllServiceRetire()
			} else {
				NodeIsRun = false
				log.Info("receipt stop signal.")
			}
		case <-pProfilerTicker.C:
			profiler.Report()
		}
	}

	//7.退出
	service.StopAllService()
	cluster.GetCluster().Stop()

	log.Info("Server is stop.")
	log.Close()
	return nil
}

type templateServicePoint[T any] interface {
	*T
	service.IService
}

func Setup(s ...service.IService) {
	for _, sv := range s {
		sv.OnSetup(sv)
		preSetupService = append(preSetupService, sv)
	}
}

func SetupTemplateFunc(fs ...func() service.IService) {
	for _, f := range fs {
		preSetupTemplateService = append(preSetupTemplateService, f)
	}
}

func SetupTemplate[T any, P templateServicePoint[T]]() {
	SetupTemplateFunc(func() service.IService {
		var t T
		return P(&t)
	})
}

func GetService(serviceName string) service.IService {
	return service.GetService(serviceName)
}

func SetConfigDir(cfgDir string) {
	configDir = cfgDir
	cluster.SetConfigDir(cfgDir)
}

func GetConfigDir() string {
	return configDir
}

func OpenProfilerReport(interval time.Duration) {
	profilerInterval = interval
}

func setLogOutputType(e string) error {
	e = strings.TrimSpace(e)
	if e == "" {
		return nil
	}
	switch e {
	case "all", "console", "file":
		log.OutputType = e
	default:
		return errors.New("unknown log output type: " + e)
	}
	return nil
}

func setLogEncoder(e string) error {
	e = strings.TrimSpace(e)
	if e == "" {
		return nil
	}
	switch e {
	case "json", "console":
		log.Encoder = e
	default:
		return errors.New("unknown log encoder: " + e)
	}
	return nil
}

func setLevel(strlogLevel string) error {
	strlogLevel = strings.TrimSpace(strlogLevel)
	if strlogLevel == "" {
		return nil
	}
	switch strlogLevel {
	case "debug":
		log.LogLevel = log.LevelDebug
	case "info":
		log.LogLevel = log.LevelInfo
	case "warning":
		log.LogLevel = log.LevelWarning
	case "error":
		log.LogLevel = log.LevelError
	case "stack":
		log.LogLevel = log.LevelStack
	case "fatal":
		log.LogLevel = log.LevelFatal
	default:
		return errors.New("unknown level: " + strlogLevel)
	}
	return nil
}

func setLogPath(e string) error {
	e = strings.TrimSpace(e)
	if e == "" {
		return nil
	}

	log.LogPath = e
	dir, err := os.Stat(log.LogPath) //这个文件夹不存在
	if err == nil && dir.IsDir() == false {
		return errors.New("Not found dir " + log.LogPath)
	}

	if err != nil {
		err = os.Mkdir(log.LogPath, os.ModePerm)
		if err != nil {
			return errors.New("Cannot create dir " + log.LogPath)
		}
	}

	return nil
}

func setLogSize(e int) error {
	if e <= 0 {
		return nil
	}
	log.LogSize = e
	return nil
}
