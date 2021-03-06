package utils

import (
	"SwitchLogFSNotify/basic"
	"SwitchLogFSNotify/models"
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"myCommon/common/kafka"
	"myCommon/common/myredis"
	"myCommon/common/utils"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

/*
** ios:
     	ntp server 10.2.8.44 prefer
     	clock timezone CST 8 0
		service timestamps log datetime msec localtime show-timezone year
		logging origin-id string NET-A07-8U-3850s(10.0.255.245)
		logging trap 7
 		logging host 10.2.13.36 transport udp port 5140
** NX-OS:
 		ntp server 10.2.8.44 prefer
		clock timezone CST 8 0
		logging timestamp milliseconds
		logging origin-id string ZH-DC1-NR-B-HDA-9372A(10.0.255.215)
		logging server 10.2.13.35
*/
const (
	level        = 3
	mistakeEvent = 22
	mStr1        = "switchport access vlan"
	mStr2        = "authentication event server dead action reinitialize vlan"
	extra        = "send by aiops"

	term1     = "ntp server 10.2.8.44 prefer"
	term2     = "clock timezone CST 8 0"
	iosTerm1  = "service timestamps log datetime msec localtime show-timezone year"
	iosTerm2  = "logging trap debugging"
	iosTerm3  = "logging host 10.2.13.36 transport udp port 5140"
	nxosTerm1 = "logging timestamp milliseconds"
	nxosTerm2 = "logging server 10.2.13.35"
	url       = "http://10.2.15.54:18081/v2/backup?url=http://network.gree.com"

	authVlan="%PORT_AUTH_VLAN_INVALID"
)

func MistakeAlarm(dir string) {
	log.Info("Begin listening......")
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error("创建文件监听器失败: ", err)
		return
	}
	defer watch.Close()
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			path, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			err = watch.Add(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
LOOP:
	for {
		select {
		case ev := <-watch.Events:
			if ev.Op&fsnotify.Create == fsnotify.Create {
				log.Info("有新的文件产生: ", ev.Name)
				f, err := os.Stat(ev.Name)
				if err != nil {
					log.Error("Error: ", err)
					return
				}
				if f.IsDir() {
					watch.Add(ev.Name)
				} else {
					var switchIp, host, port, vlan1, vlan2 string

					syslog := make(map[string]bool, 0)
					strs := strings.Split(ev.Name, "/")
					for _, v := range strs {
						reg := regexp.MustCompile(`(2(5[0-5]{1}|[0-4]\d{1})|[0-1]?\d{1,2})(\.(2(5[0-5]{1}|[0-4]\d{1})|[0-1]?\d{1,2})){3}`)
						if reg.MatchString(v) {
							switchIp = v
						}
					}
					fmt.Println("Switch IP: ", switchIp)
					key := basic.RC.SetKey + ":" + switchIp
					fileName := strs[len(strs)-1]
					//tm := f.ModTime().UnixNano() / 1e6
					tm:=f.ModTime()
					mTime := f.ModTime().Format(basic.TimeStr)

					// 1.更新最新的文件的产生时间
					var id int
					var lastTime string
					db := basic.DB.Raw("select id from switch_log_record where switch_ip=?", switchIp)
					db.Row().Scan(&id)
					if id != 0 {
						db = basic.DB.Raw("select last_time from switch_log_record where id=?", id)
						db.Row().Scan(&lastTime)

						if lastTime < mTime {
							fmt.Println("最新时间: ",mTime)
							db = basic.DB.Exec("update switch_log_record set last_time=? where id=?",
								mTime,id)
							fmt.Println("[update]Rows affected: ", db.RowsAffected)
						}
					} else {
						db = basic.DB.Exec("insert into switch_log_record set last_time=? ,switch_ip=?",
							mTime, switchIp)
						if err:=db.Error;err!=nil{
							log.Error("新增数据失败: ",err)
							goto LOOP
						}
						fmt.Println("[insert]Rows affected: ", db.RowsAffected)
					}

					// 2.将产生的文件上传到HDFS
					var name string
					if strings.Contains(ev.Name, "network") {
						str := strings.Split(ev.Name, "network")
						name = str[1]
					} else if strings.Contains(ev.Name, "support") {
						str := strings.Split(ev.Name, "support")
						name = "/support" + str[1]
					}
					uri := url + name
					request, err := http.NewRequest("GET", uri, strings.NewReader(""))
					if err != nil {
						log.Error("Request failed: ", err)
						return
					}
					request.Header.Add("Content-Type", "application/json")

					client := &http.Client{}
					response, err := client.Do(request)

					if err != nil {
						log.Error("Client do failed: ", err)
						return
					}
					res, err := ioutil.ReadAll(response.Body)
					if err != nil {
						log.Error("Get body failed: ", err)
						return
					}
					log.Info("Response body: ", string(res))

					// 3.判断是否有配置错漏，发送告警，存入Redis
					time.Sleep(200 * time.Millisecond)
					file, err := os.Open(ev.Name)
					if err != nil {
						log.Error("打开文件失败: ", err)
						goto LOOP
					}
					//mp := make(map[string][]string, 0)
					data := []interface{}{}
					utils.InterfaceSliceClear1(&data)
					scanner := bufio.NewScanner(file)
					for scanner.Scan() {
						mp := make(map[string]interface{}, 0)
						vlan := make(map[string]string)
						//var vlans []string
						line := scanner.Text()
						if strings.Contains(line, "hostname") {
							str := strings.Split(line, " ")
							host = str[len(str)-1]
						}
						if strings.Contains(line, "Ethernet") {
							str := strings.Split(line, " ")
							port = str[len(str)-1]
						}
						if strings.Contains(line, mStr1) {
							str := strings.Split(line, " ")
							vlan1 = str[len(str)-1]
							vlan["vlan1"] = vlan1
							mp[port] = vlan
						}
						if strings.Contains(line, mStr2) {
							str := strings.Split(line, " ")
							vlan2 = str[len(str)-1]
							vlan["vlan2"] = vlan2
							mp[port] = vlan
						}
						if strings.Contains(line, "logging origin-id string") {
							str := strings.Split(line, " ")
							ts := host + "(" + switchIp + ")"
							fmt.Println("Target: ", ts)
							if str[len(str)-1] == ts {
								syslog[line] = true
							} else {
								syslog[line] = false
							}
						}
						if line == term1 {
							syslog[line] = true
						}

						if line == term2 {
							syslog[line] = true
						}

						if line == iosTerm1 {
							syslog[line] = true
						}
						if strings.Contains(line, iosTerm2) {
							syslog[line] = true
						}
						if line == iosTerm3 {
							syslog[line] = true
						}
						if line == nxosTerm1 {
							syslog[line] = true
						}
						if strings.Contains(line, nxosTerm2) {
							syslog[line] = true
						}
						for k, v := range mp {
							if v != nil {
								var sv models.SwitchVlan
								sv.Port = k
								sv.Vlan = v
								data = append(data, sv)
							}
						}
					}
					var switchWarm []models.SwitchWarn
					for i := 0; i < len(data); i++ {
						var sw models.SwitchWarn
						for j := i + 1; j < len(data); j++ {
							port1 := data[i].(models.SwitchVlan).Port
							port2 := data[j].(models.SwitchVlan).Port
							if port1 != "" && port2 != "" && port1 == port2 {
								v1 := data[i].(models.SwitchVlan).Vlan.(map[string]string)
								v2 := data[j].(models.SwitchVlan).Vlan.(map[string]string)
								for k1, v := range v1 {
									for k2, vv := range v2 {
										if k1 == "vlan1" && k2 == "vlan2" && v != vv {
											sw.Port = port1
											sw.AccessVlan, _ = strconv.Atoi(v)
											sw.AuthVlan, _ = strconv.Atoi(vv)
											switchWarm = append(switchWarm, sw)
										}
									}
								}
							}
						}
					}
					//for k, vv := range mp {
					//	var sw models.SwitchWarn
					//	if k != "" && vv[0] != "" && vv[1] != "" && vv[0] != vv[1] {
					//		// 告警
					//		//alarm.Tm = time.Now().UTC().UnixNano() / 1e6
					//		//alarm.Msg = fmt.Sprintf("交换机 %s(%s):%s: 配置文件中端口的配置信息错漏 (%s)",
					//		//	host, switchIp, fileName, extra)
					//		//origin = origin + "\n" + fmt.Sprintf("交换机 %s(%s):%s 的配置文件中(%s)端口准入认证配置的vlan:%s和access允许通过的vlan:%s不一致",
					//		//	host, switchIp, fileName, k, vv[0], vv[1])
					//		//ports = ports + " " + k
					//		sw.Port = k
					//		sw.Vlan1, _ = strconv.Atoi(vv[0])
					//		sw.Vlan2, _ = strconv.Atoi(vv[1])
					//		switchWarm = append(switchWarm, sw)
					//	}
					//}
					//fmt.Println(switchWarm)
					if len(switchWarm) != 0 {
						var ports,content,topic string
						for _,v:=range switchWarm{
							ports=ports+" "+v.Port
						}
						var ext models.Ext
						ext.ConfVlans=switchWarm
						js,err:=json.Marshal(ext)
						//fmt.Println(string(js))
						if err!=nil{
							log.Error("数据转换失败: ",err)
							goto LOOP
						}
						hostIp:=host+"("+switchIp+")"
						getKey:=basic.RC.GetKey+switchIp
						r,err:=myredis.RedisClient.Get(getKey).Result()
						if err!=nil{
							log.Error("[Redis]Get system type error: ",err)
							goto LOOP
						}
						sys,_:=strconv.Atoi(r)
						if sys == 0 {
							log.Info("System type is unknown!")
							goto LOOP
						}else if sys == 1{
							// NX-OS
							topic=basic.KC.Topic
							content=fmt.Sprintf("<186>%s:%s:%s:交换机 %s 的配置文件 %s 中 %s " +
								"端口准入认证配置的vlan和access允许通过的vlan不一致 %s - from 智能运维平台",
								hostIp,tm,authVlan,hostIp,fileName,ports,string(js))
						}else if sys == 2{
							// IOS
							topic=basic.KC.Topic+"_ios"
							content=fmt.Sprintf("<189>%s:%s:%s:交换机 %s 的配置文件 %s 中 %s " +
								"端口准入认证配置的vlan和access允许通过的vlan不一致 %s - from 智能运维平台",
								hostIp,tm,authVlan,hostIp,fileName,ports,string(js))
						}

						//var alarm models.Warning
						//alarm.Tm = tm
						//alarm.Title = "配置错漏告警"
						//db:=basic.DB.Raw("select switch_category from net_switch where ip=?",switchIp)
						//db.Row().Scan(&alarm.Category)
						////alarm.Category = category
						//alarm.Level = level
						//alarm.Ip = switchIp
						//alarm.Host = host
						//alarm.Event = mistakeEvent
						//alarm.Msg = fmt.Sprintf("交换机 %s(%s):%s: 配置文件中端口的配置信息错漏 (%s)",
						//	host, switchIp, fileName, extra)
						//alarm.Origin = ""
						////alarm.Origin = origin+" "+extra
						//alarm.Ext = models.Ext{ConfVlans: switchWarm}
						//js, err := json.Marshal(alarm)
						//if err != nil {
						//	log.Error("数据转换失败: ", err)
						//	return
						//}
						// 发送错误告警 omitempty
						fmt.Println("配置错漏告警")
						kafka.SyncProducer(basic.KC.Broker, topic, basic.KC.Key, content)
					}

					for i := 0; i < len(data); i++ {
						for j := i + 1; j < len(data); j++ {
							port1 := data[i].(models.SwitchVlan).Port
							port2 := data[j].(models.SwitchVlan).Port
							if port1 != "" && port2 != "" && port1 == port2 {
								v1 := data[i].(models.SwitchVlan).Vlan.(map[string]string)
								v2 := data[j].(models.SwitchVlan).Vlan.(map[string]string)
								for k1, v := range v1 {
									for k2, vv := range v2 {
										if k1 == "vlan1" && k2 == "vlan2" && v != vv {
											err = myredis.RedisClient.Set(key, 0, 0).Err()
											if err != nil {
												log.Error("[Redis]写入失败: ", err)
												return
											}
											log.Info("[Redis]Vlan 配置有误,写入成功: ", key)
											goto LOOP
										}
									}
								}
							}
						}
					}
					//for k, v := range mp {
					//	if k != "" && v[0] != "" && v[1] != "" && v[0] != v[1] {
					//		// syslog available 为 false 0
					//		err = myredis.RedisClient.Set(key, 0, 0).Err()
					//		if err != nil {
					//			log.Error("[Redis]写入失败: ", err)
					//			return
					//		}
					//		log.Info("[Redis]Vlan 配置有误,写入成功: ", key)
					//		goto LOOP
					//	}
					//}

					// 没有以上错误,判断其他内容是否有误
					//fmt.Println("SysLog: ", syslog)
					for _, v := range syslog {
						fmt.Println("Syslog: ", v)
						if v == false {
							err := myredis.RedisClient.Set(key, 0, 0).Err()
							if err != nil {
								log.Error("[Redis]写入失败: ", err)
								return
							}
							log.Info("[Redis]系统配置有误，写入成功: ", key)
							goto LOOP
						}
					}
					// 所有内容都无误,则为 true(1)
					//key = key + ":" + switchIp
					err = myredis.RedisClient.Set(key, 1, 0).Err()
					if err != nil {
						log.Error("[Redis]写入失败: ", err)
						return
					}
					log.Info("[Redis]系统配置无误，写入成功: ", key)

				} // else
			}
		case err := <-watch.Errors:
			log.Error("Error: ", err)
			break LOOP
		}
	}
}
