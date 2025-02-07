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
	loadSecConf(conn)
}

//加载秒杀商品信息
func loadSecConf(conn *zk.Conn) {
	log.Printf("Connect zk sucess %s", conf.Zk.SecProductKey)
	v, _, err := conn.Get(conf.Zk.SecProductKey) //conf.Etcd.EtcdSecProductKey
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

//更新秒杀商品信息
func updateSecProductInfo(secProductInfo []*conf.SecProductInfoConf) {
	tmp := make(map[int]*conf.SecProductInfoConf, 1024)
	for _, v := range secProductInfo {
		log.Printf("updateSecProductInfo %v", v)
		tmp[v.ProductId] = v
	}
	conf.SecKill.RWBlackLock.Lock()
	conf.SecKill.SecProductInfoMap = tmp
	conf.SecKill.RWBlackLock.Unlock()
}
