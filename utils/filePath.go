package utils

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"regexp"
	"strings"
)

const (
	listFilePrefix = " "
)

func FilePathList(level int, separator, fileDir string) ([]string,error) {
	var filePath []string
	//fileInfos := []interface{}{}
	//utils.InterfaceSliceClear1(&fileInfos)
	files, err := ioutil.ReadDir(fileDir)
	if err != nil {
		log.Error("读取目录失败: ", err)
		return nil, err
	}
	tmpPrefix := ""
	for i := 1; i < level; i++ {
		tmpPrefix = tmpPrefix + listFilePrefix
	}

	for _, one := range files {
		//log.Println("123456: ", fileDir+path+one.Name())
		if one.IsDir() {
			// 根目录
			// 二级目录，二级目录对应交换机的 IP 地址
			//fmt.Printf("\033[34m %s %s \033[0m \n", tmpPrefix, one.Name())
			FilePathList(level+1, separator, fileDir+separator+one.Name())
			//FileListInfo()
			//dirMap[one.Name()] = one.Name()
		} else {
			//filePath:=fileDir+path+one.Name()
			//log.Println("123456: ", fileDir)
			//log.Println(tmpPrefix, " ", one.Name(), " ", one.ModTime(), " ", one.Size())
			//var switchIp string
			str := strings.Split(fileDir, "/")
			reg := regexp.MustCompile(`(2(5[0-5]{1}|[0-4]\d{1})|[0-1]?\d{1,2})(\.(2(5[0-5]{1}|[0-4]\d{1})|[0-1]?\d{1,2})){3}`)
			if reg.MatchString(str[len(str)-1]) {
				//fmt.Println("IP地址: ", str[len(str)-1])
				filePath=append(filePath,fileDir)
			}else{
				continue
			}

		}
	}
	return filePath,nil
}