package cmd

import (
	"context"
	"fmt"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/zh-five/xdaemon"
	"go-fly-muti/common"
	"go-fly-muti/controller"
	"go-fly-muti/logger"
	"go-fly-muti/middleware"
	"go-fly-muti/models"
	"go-fly-muti/router"
	"go-fly-muti/setting"
	"go-fly-muti/static"
	"go-fly-muti/tools"
	"go-fly-muti/ws"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var (
	port     string
	daemon   bool
	rootPath string
)
var serverCmd = &cobra.Command{
	Use:     "server",
	Short:   "启动客服http服务",
	Example: "go-fly server",
	Run:     run,
}

func init() {
	serverCmd.PersistentFlags().StringVarP(&rootPath, "rootPath", "r", "", "程序根目录")
	serverCmd.PersistentFlags().StringVarP(&port, "port", "p", "8081", "监听端口号")
	serverCmd.PersistentFlags().BoolVarP(&daemon, "daemon", "d", false, "是否为守护进程模式")
}
func run(cmd *cobra.Command, args []string) {

	log.Println("565544")
	//初始化目录
	initDir()
	//初始化守护进程
	initDaemon()

	baseServer := "0.0.0.0:" + port

	//if common.RpcStatus {
	//	go frpc.NewRpcServer(common.RpcServer)
	//	log.Println("start rpc server...\r\ngo：tcp://" + common.RpcServer)
	//}
	//加载配置
	if err := setting.Init(); err != nil {
		fmt.Println("配置文件初始化事变", err)
		return
	}

	//注册日志

	err := logger.Init()
	if err != nil {
		fmt.Println("日志err", err)
		return
	}
	defer logger.Sync()

	////设置时区
	//loc, err := time.LoadLocation(viper.GetString("app.timeZone"))
	//if err == nil {
	//	time.Local = loc // -> this is setting the global timezone
	//	fmt.Println(time.Now().Format("2006-01-02 15:04:05 "))
	//}

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(logger.GinLogger(), logger.GinRecovery(true))

	engine.MaxMultipartMemory = 32 << 20 // 32 MiB

	//是否编译模板
	if common.IsCompireTemplate {
		templ := template.Must(template.New("").ParseFS(static.TemplatesEmbed, "templates/**/*.html"))
		engine.SetHTMLTemplate(templ)
	} else {
		engine.LoadHTMLGlob(common.StaticDirPath + "templates/**/*")
	}
	engine.Static("/static", common.StaticDirPath)
	engine.Use(tools.Session("gofly"))
	//跨域设置
	engine.Use(middleware.CrossSite)

	router.InitViewRouter(engine)
	router.InitApiRouter(engine)

	//限流类
	tools.NewLimitQueue()
	//清理
	//ws.CleanVisitorExpire()
	//后端websocket
	go ws.WsServerBackend()
	//初始化数据
	//logger := lib.NewLogger()
	if err := models.NewConnect(common.ConfigDirPath + "/mysql.json"); err != nil {
		log.Printf("数据库初始化失败: %v", err)
		return
	}
	defer models.CloseDB()
	//初始化配置数据
	models.InitConfig()
	systemTimezone := models.FindConfig("SystemTimezone")
	if systemTimezone == "" {
		systemTimezone = setting.DefaultTimezone
		if err := models.SaveConfig("系统时区", "SystemTimezone", systemTimezone); err != nil {
			log.Printf("初始化系统时区配置失败: %v", err)
		}
	}
	if err := setting.ConfigureTimezone(systemTimezone); err != nil {
		log.Printf("系统时区配置无效，回退到 %s: %v", setting.DefaultTimezone, err)
		_ = setting.ConfigureTimezone(setting.DefaultTimezone)
	}
	log.Printf("系统时区: %s", setting.CurrentTimezone())
	//后端定时客服
	go ws.UpdateVisitorStatusCron()

	//定时检查客服是否有未读信息
	//go process.CheckUnreadMes()

	log.Println("GOFLY服务开始运行:" + baseServer)
	// 性能监控默认关闭，避免在生产环境暴露运行时信息。
	if strings.EqualFold(os.Getenv("GOFLY_ENABLE_PPROF"), "true") {
		pprof.Register(engine)
	}
	//engine.Run(baseServer)

	srv := &http.Server{
		Addr:    baseServer,
		Handler: engine,
	}

	serverErrors := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignals)
	receivedSignal, serverErr := waitForShutdown(controller.StopSign, shutdownSignals, serverErrors)
	if serverErr != nil {
		log.Printf("GOFLY服务监听失败: %v", serverErr)
		return
	}
	if receivedSignal != nil {
		log.Printf("收到系统信号 %s，开始关闭服务", receivedSignal)
	}
	log.Println("关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("服务关闭失败:", err)
	}
	log.Println("服务已关闭")

}

func waitForShutdown(stopSign <-chan int, signals <-chan os.Signal, serverErrors <-chan error) (os.Signal, error) {
	select {
	case <-stopSign:
		return nil, nil
	case receivedSignal := <-signals:
		return receivedSignal, nil
	case err := <-serverErrors:
		return nil, err
	}
}

// 初始化目录
func initDir() {
	if rootPath == "" {
		rootPath = tools.GetRootPath()
	}
	log.Println("GOFLY服务运行路径:" + rootPath)
	common.RootPath = rootPath
	common.LogDirPath = rootPath + "/logs/"
	common.ConfigDirPath = rootPath + "/config/"
	common.StaticDirPath = rootPath + "/static/"
	common.UploadDirPath = rootPath + "/static/upload/"

	common.KFMYArray = strings.Split(common.KFMY, "\n")

	if noExist, _ := tools.IsFileNotExist(common.RootPath + "/install.lock"); noExist {
		panic("未检测到" + common.RootPath + "/install.lock,请先安装服务!")

	}

	if noExist, _ := tools.IsFileNotExist(common.LogDirPath); noExist {
		if err := os.MkdirAll(common.LogDirPath, 0777); err != nil {
			log.Println(err.Error())
		}
	}
	isMainUploadExist, _ := tools.IsFileExist(common.UploadDirPath)
	if !isMainUploadExist {
		os.Mkdir(common.UploadDirPath, os.ModePerm)
	}
}

// 初始化守护进程
func initDaemon() {
	pidPath := filepath.Join(common.RootPath, pidFileName)
	// xdaemon 会让同一命令依次作为启动器、守护父进程和工作进程执行。
	// 只允许最外层启动器清理旧实例，避免守护父进程误停刚启动的子进程。
	if os.Getenv(xdaemon.ENV_NAME) == "" {
		if err := stopProcesses(pidPath); err != nil {
			log.Fatalf("停止旧服务失败: %v", err)
		}
	}

	if daemon {
		d := xdaemon.NewDaemon(common.LogDirPath + "gofly.log")
		d.MaxError = 10
		d.Run()
	}

	pids := []int{os.Getpid()}
	if daemon {
		pids = []int{os.Getppid(), os.Getpid()}
	}
	if err := writeProcessIDs(pidPath, pids...); err != nil {
		log.Fatalf("写入 PID 文件失败: %v", err)
	}
}
