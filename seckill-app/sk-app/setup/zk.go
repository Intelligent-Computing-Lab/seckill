package setup

import (
	"encoding/json"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"log"
	conf "seckill/pkg/config"
	"time"
)

//初始化Etcd
func InitZk() {
	var hosts = []string{"49.233.111.31:2181"}
	//option := zk.WithEventCallback(waitSecProductEvent)
	conn, _, err := zk.Connect(hosts, time.Second*5)
	if err != nil {
		fmt.Println(err)
		return
	}

	conf.Zk.ZkConn = conn
	conf.Zk.SecProductKey = "/product"
	//加载秒杀商品信息
	loadSecConf(conn)
}

//加载秒杀商品信息
func loadSecConf(conn *zk.Conn) {
	log.Printf("Connect zk sucess %s", conf.Zk.SecProductKey)
	v, _, err := conn.Get(conf.Zk.SecProductKey) //conf.Etcd.EtcdSecProductKey
	print("姜猛修改标记，打印获取到的v", v)
	if err != nil {
		log.Printf("get product info failed, err : %v", err)
		return
	}
	log.Printf("get product info ")
	var secProductInfo []*conf.SecProductInfoConf
	err1 := json.Unmarshal(v, &secProductInfo)
	if err1 != nil {
		log.Printf("Unmsharl second product info failed, err : %v", err1)
	}
	print("姜猛修改标记，打印传的参数secProductInfo", secProductInfo)
	updateSecProductInfo(secProductInfo)
}

func waitSecProductEvent(event zk.Event) {
	log.Print(">>>>>>>>>>>>>>>>>>>")
	log.Println("path:", event.Path)
	log.Println("type:", event.Type.String())
	log.Println("state:", event.State.String())
	log.Println("<<<<<<<<<<<<<<<<<<<")
	if event.Path == conf.Zk.SecProductKey {

	}
}

//监听秒杀商品配置
//for wrsp := range rch {
//	for _, ev := range wrsp.Events {
//		//删除事件
//		if ev.Type == mvccpb.DELETE {
//			continue
//		}
//
//		//更新事件
//		if ev.Type == mvccpb.PUT && string(ev.Kv.Key) == key {
//			err := json.Unmarshal(ev.Kv.Value, &secProductInfo)
//			if err != nil {
//				getConfSucc = false
//				continue
//			}
//		}
//	}
//
//	if getConfSucc {
//		updateSecProductInfo(secProductInfo)
//	}
//}

//更新秒杀商品信息；监听秒杀商品配置
func updateSecProductInfo(secProductInfo []*conf.SecProductInfoConf) {
	tmp := make(map[int]*conf.SecProductInfoConf, 1024)
	fmt.Println("姜猛修改标记，打印存入SecProductInfoMap的tmp", tmp)
	for _, v := range secProductInfo {
		log.Printf("updateSecProductInfo %v", v)
		tmp[v.ProductId] = v
	}
	conf.SecKill.RWBlackLock.Lock()
	conf.SecKill.SecProductInfoMap = tmp
	conf.SecKill.RWBlackLock.Unlock()
}
