package cmd

import (
	"github.com/spf13/cobra"
	"go-fly-muti/common"
	"go-fly-muti/models"
	"go-fly-muti/tools"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "安装导入数据",
	Run: func(cmd *cobra.Command, args []string) {
		install()
	},
}

func install() {
	if ok, _ := tools.IsFileNotExist("./install.lock"); !ok {
		log.Println("请先删除./install.lock")
		os.Exit(1)
	}
	sqlFile := common.Dir + "go-fly.sql"
	isExit, _ := tools.IsFileExist(common.MysqlConf)
	dataExit, _ := tools.IsFileExist(sqlFile)
	if !isExit || !dataExit {
		log.Println("config/mysql.json 数据库配置文件或者数据库文件go-fly.sql不存在")
		os.Exit(1)
	}
	if err := models.NewConnect(common.MysqlConf); err != nil {
		log.Printf("数据库初始化失败: %v", err)
		os.Exit(1)
	}
	defer models.CloseDB()
	sqls, _ := ioutil.ReadFile(sqlFile)
	sqlArr := strings.Split(string(sqls), "|")
	for _, sql := range sqlArr {
		if sql == "" {
			continue
		}
		err := models.Execute(sql)
		if err == nil {
			log.Println(sql, "\t success!")
		} else {
			log.Println(sql, err, "\t failed!")
			os.Exit(1)
		}
	}
	if _, err := models.RunSchemaMigrations(models.DB); err != nil {
		log.Printf("安装后数据库迁移失败: %v", err)
		os.Exit(1)
	}
	installFile, _ := os.OpenFile("./install.lock", os.O_RDWR|os.O_CREATE, os.ModePerm)

	installTime := tools.Int64ToByte(time.Now().Unix())
	token, err := tools.AesEncrypt(installTime, []byte(common.AesKey))
	if err != nil {
		log.Println(err)
	}
	installFile.Write(token)
}
