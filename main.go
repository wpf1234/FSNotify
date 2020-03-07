package main

import (
	"SwitchLogFSNotify/basic"
	"SwitchLogFSNotify/utils"
	"github.com/gin-gonic/gin"
	"github.com/micro/go-micro/web"
	log "github.com/sirupsen/logrus"
	"myCommon/common/middleware"
)

func main() {
	service := web.NewService(
		web.Name("switchLog"),
		web.Version("latest"),
	)
	service.Init()
	basic.Init()

	//go func() {
	//	for {
	//		now := time.Now()
	//		utils.FileNotify(basic.SC.Path)
	//		// 计算下一个零点
	//		next := now.Add(time.Hour * 24)
	//		next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
	//		t := time.NewTimer(next.Sub(now))
	//		<-t.C
	//		//utils.FileNotify(basic.SC.Path)
	//	}
	//}()

	//go utils.FileNotify(basic.SC.Path)

	go utils.MistakeAlarm(basic.SC.Path)

	gin.SetMode(gin.ReleaseMode) // 全局设置环境，debug 为开发环境，线上环境为 gin.ReleaseMode
	router := gin.Default()
	router.Use(middleware.Cors())

	// 注册 handler
	service.Handle("/", router)

	// 启动 API
	err := service.Run()
	if err != nil {
		log.Error("服务启动失败: ", err)
		return
	}
}
