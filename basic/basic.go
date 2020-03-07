package basic

import (
	"SwitchLogFSNotify/models"
	"encoding/base64"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/micro/go-micro/config"
	"github.com/micro/go-micro/config/source/file"
	"myCommon/common/logger"
	"myCommon/common/myredis"
	"myCommon/common/utils"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	_ "github.com/go-sql-driver/mysql"
)

const (
	defaultRoot = "app"
	TimeStr     = "2006-01-02 15:04:05"
	Level       = 1
)

var (
	m         sync.RWMutex
	Key       = []byte("aiops*2019*1801.")
	SC        models.SwitchConf
	LC        models.LogConf
	RC        models.RedisConf
	KC        models.KafkaConf
	DB        *gorm.DB
	mc        models.MysqlConf
	Separator = string(os.PathSeparator)
)

func Init() {
	m.Lock()
	defer m.Unlock()

	err := config.Load(file.NewSource(
		file.WithPath("./conf/application.yml"),
	))
	if err != nil {
		log.Error("加载配置文件失败: ", err)
		return
	}
	if err := config.Get(defaultRoot, "log").Scan(&LC); err != nil {
		log.Error("Log 配置文件读取失败: ", err)
		return
	}
	log.Info("读取 Log 配置成功!")
	logger.LoggerToFile(LC.Path, LC.File)

	if err := config.Get(defaultRoot, "switch").Scan(&SC); err != nil {
		log.Error("Switch 配置文件读取失败: ", err)
		return
	}
	log.Info("读取 Switch 配置成功!")

	if err := config.Get(defaultRoot, "kafka").Scan(&KC); err != nil {
		log.Error("Kafka 配置文件读取失败: ", err)
		return
	}
	log.Info("读取 Kafka 配置成功!")

	if err := config.Get(defaultRoot, "redis").Scan(&RC); err != nil {
		log.Error("Redis 配置文件读取失败: ", err)
		return
	}
	log.Info("读取 Redis 配置成功!")

	redisStr, err := base64.StdEncoding.DecodeString(RC.Password)
	if err != nil {
		log.Error("Base64 decode failed: ", err)
		return
	}
	redisPwd, err := utils.AesDecrypt(redisStr, Key)
	if err != nil {
		log.Error("ASE decrypt failed: ", err)
		return
	}
	err = myredis.NewClient(RC.Host, string(redisPwd), RC.Port, RC.DB, RC.Pool)
	if err != nil {
		log.Error("创建 redis client 失败: ", err)
		return
	}

	log.Info("链接 redis 成功!")

	if err := config.Get(defaultRoot, "mysql").Scan(&mc); err != nil {
		log.Error("Mysql 配置文件读取失败: ", err)
		return
	}
	log.Info("读取 Mysql 配置成功!")

	str, err := base64.StdEncoding.DecodeString(mc.Pwd)
	if err != nil {
		log.Error("Base64 decode failed: ", err)
		return
	}
	pwd, err := utils.AesDecrypt(str, Key)
	if err != nil {
		log.Error("ASE decrypt failed: ", err)
		return
	}
	dsn := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8&parseTime=true",
		mc.User, string(pwd), mc.Host, mc.Db)
	DB, err = gorm.Open("mysql", dsn)
	if err != nil {
		log.Error("Open mysql failed: ", err)
		return
	}

	DB.DB().SetMaxIdleConns(10)
	DB.DB().SetMaxOpenConns(10)
	DB.DB().SetConnMaxLifetime(10 * time.Second)

	if err := DB.DB().Ping(); err != nil {
		log.Error("连接数据库失败: ", err)
		return
	}
	log.Info("数据库连接成功!")
}
