package service

import (
	"encoding/json"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"log"
	conf "seckill/pkg/config"
	"seckill/sk-app/config"
	"seckill/sk-app/model"
	"seckill/sk-app/service/srv_err"
	"seckill/sk-app/service/srv_limit"
	"time"
)

// Service Define a service interface
type Service interface {
	// HealthCheck check service health status
	HealthCheck() bool
	SecInfo(productId int) (date map[string]interface{})
	SecKill(req *model.SecRequest) (map[string]interface{}, int, error)
	SecInfoList() ([]map[string]interface{}, int, error)
}

//UserService implement Service interface
type SkAppService struct {
}

// HealthCheck implement Service method
// 用于检查服务的健康状态，这里仅仅返回true
func (s SkAppService) HealthCheck() bool {
	return true
}

type ServiceMiddleware func(Service) Service

func (s SkAppService) SecInfo(productId int) (date map[string]interface{}) {
	//姜猛修改标记，测试添加个初始化zk函数
	InitZk()
	config.SkAppContext.RWSecProductLock.RLock()
	defer config.SkAppContext.RWSecProductLock.RUnlock()
	fmt.Println("我要根据商品检索商品信息id")
	fmt.Println("productId: ", productId)
	fmt.Println("姜猛修改标记，打印SecProductInfoMap的内容是", conf.SecKill.SecProductInfoMap)
	fmt.Println("zk商品数据是：", conf.SecKill.SecProductInfoMap[productId])
	v, ok := conf.SecKill.SecProductInfoMap[productId]
	if !ok {
		fmt.Println("ok 是false")
		return nil
	}

	data := make(map[string]interface{})
	data["product_id"] = productId
	data["start_time"] = v.StartTime
	data["end_time"] = v.EndTime
	data["status"] = v.Status
	fmt.Println("SecInfo要返回的数据是：", data)
	return data
}

func (s SkAppService) SecKill(req *model.SecRequest) (map[string]interface{}, int, error) {
	fmt.Println("姜猛修改标记，开始调用秒杀的service方法SecKill")
	//姜猛修改标记，测试添加个初始化zk函数
	InitZk()
	//对Map加锁处理
	//config.SkAppContext.RWSecProductLock.RLock()
	//defer config.SkAppContext.RWSecProductLock.RUnlock()
	var code int
	//1。黑名单校验，2。流量限制
	log.Printf("req is: ", req)
	err := srv_limit.AntiSpam(req)
	if err != nil {
		code = srv_err.ErrUserServiceBusy
		log.Printf("userId antiSpam [%d] failed, req[%v]", req.UserId, err)
		return nil, code, err
	}
	//3。获取商品信息
	data, code, err := SecInfoById(req.ProductId)
	//当前错误是没找到product_id是
	if err != nil {
		log.Printf("获取商品信息出错，原因是： ", err)
		log.Printf("userId[%d] secInfoById Id failed, req[%v]", req.UserId, req)
		return nil, code, err
	}
	//4。把请求推入到SecReqChan，该请求会经过redis队列，最终呗core处理，并经过另一个redis队列发送到resultChan
	userKey := fmt.Sprintf("%d_%d", req.UserId, req.ProductId)
	log.Printf("userKey是", userKey)
	ResultChan := make(chan *model.SecResult, 1)
	config.SkAppContext.UserConnMapLock.Lock()
	config.SkAppContext.UserConnMap[userKey] = ResultChan
	config.SkAppContext.UserConnMapLock.Unlock()

	//将请求送入通道并推入到redis队列当中
	config.SkAppContext.SecReqChan <- req

	ticker := time.NewTicker(time.Millisecond * time.Duration(conf.SecKill.AppWaitResultTimeout))
	//定时器
	defer func() {
		ticker.Stop()
		config.SkAppContext.UserConnMapLock.Lock()
		delete(config.SkAppContext.UserConnMap, userKey)
		config.SkAppContext.UserConnMapLock.Unlock()
	}()
	//利用select语句，进行不同结果的响应
	select {
	case <-ticker.C:
		code = srv_err.ErrProcessTimeout
		err = fmt.Errorf("request timeout")
		return nil, code, err
	case <-req.CloseNotify:
		code = srv_err.ErrClientClosed
		err = fmt.Errorf("client already closed")
		return nil, code, err
	case result := <-ResultChan:
		code = result.Code
		if code != 1002 {
			return data, code, srv_err.GetErrMsg(code)
		}
		log.Printf("secKill success")
		data["product_id"] = result.ProductId
		data["token"] = result.Token
		data["user_id"] = result.UserId
		return data, code, nil
	}
}

func NewSecRequest() *model.SecRequest {
	secRequest := &model.SecRequest{
		ResultChan: make(chan *model.SecResult, 1),
	}
	return secRequest
}

func (s SkAppService) SecInfoList() ([]map[string]interface{}, int, error) {
	//姜猛修改标记，测试添加个初始化zk函数
	InitZk()
	config.SkAppContext.RWSecProductLock.RLock()
	defer config.SkAppContext.RWSecProductLock.RUnlock()
	var data []map[string]interface{}
	for _, v := range conf.SecKill.SecProductInfoMap {
		log.Printf("v.ProductId是：  ", v.ProductId)
		//逐个根据商品id获取商品数据
		item, _, err := SecInfoById(v.ProductId)
		if err != nil {
			log.Printf("get sec info, err : %v", err)
			continue
		}
		data = append(data, item)
	}
	log.Printf("data是： ", data)
	return data, 0, nil
}

func SecInfoById(productId int) (map[string]interface{}, int, error) {
	//对Map加锁处理
	//config.SkAppContext.RWSecProductLock.RLock()
	//defer config.SkAppContext.RWSecProductLock.RUnlock()
	log.Printf("我要开始根据商品id： %s 查询商品信息了", productId)
	var code int
	v, ok := conf.SecKill.SecProductInfoMap[productId]

	if !ok {
		return nil, srv_err.ErrNotFoundProductId, fmt.Errorf("not found product_id:%d", productId)
	}
	start := false      //秒杀活动是否开始
	end := false        //秒杀活动是否结束
	status := "success" //状态
	var err error
	nowTime := time.Now().Unix()
	//秒杀活动没有开始
	if nowTime-v.StartTime < 0 {
		start = false
		end = false
		status = "second kill not start"
		code = srv_err.ErrActiveNotStart
		err = fmt.Errorf(status)
		fmt.Println("商品秒杀活动还没开始")
	}

	//秒杀活动已经开始
	if nowTime-v.StartTime > 0 {
		start = true
		fmt.Println("商品秒杀活动已经开始")
	}

	//秒杀活动已经结束
	if nowTime-v.EndTime > 0 {
		start = false
		end = true
		status = "second kill is already end"
		code = srv_err.ErrActiveAlreadyEnd
		err = fmt.Errorf(status)
		fmt.Println("商品秒杀活动已经结束")
	}

	//商品已经被停止或售磬
	if v.Status == config.ProductStatusForceSaleOut || v.Status == config.ProductStatusSaleOut {
		start = false
		end = false
		status = "product is sale out"
		code = srv_err.ErrActiveSaleOut
		err = fmt.Errorf(status)
		fmt.Println("商品商品已经被停止或售磬")
	}

	//curRate := rand.Float64()
	/**
	 * 放大于购买比率的1.5倍的请求进入core层
	 */
	//----------------------注释代码
	//if curRate > v.BuyRate*1.5 {
	//	fmt.Println("放请求进core层")
	//	start = false
	//	end = false
	//	status = "retry"
	//	code = srv_err.ErrRetry
	//	err = fmt.Errorf(status)
	//}

	//组装数据
	data := map[string]interface{}{
		"product_id": productId,
		"start":      start,
		"end":        end,
		"status":     status,
	}
	fmt.Println("商品数据为： ", data)
	return data, code, err
}

//姜猛修改标记,整体复制zk.go的函数,分割线+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//初始化Etcd
func InitZk() {
	var hosts = []string{"152.136.197.135:2181"}
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
