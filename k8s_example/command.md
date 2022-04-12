查看资源版本标签
kubectl api-versions

创建资源对象
kubectl create -f nginx.yaml

查看创建的service
kubectl get svc
kubectl describe svc svcName
查看创建的pod资源
kubectl get pods -o wide

在kubernetes查询Node IP
1.kubectl get nodes 
2.kubectl describe node nodeName  

在kubernetes查询Pod IP
1.kubectl get pods
2.kubectl describe pod podName

查看service的url
minikube service --url svcName

进容器
kubectl exec -it consul-server-0 -- /bin/sh

查看pod日志
kubectl logs podName

查看rc节点
kubectl get rc

新建namespace
kubectl create namespace seckill

进数据库看
mysql -uroot -p

dockerfile打包镜像
docker build -t runoob/ubuntu:v1 . 

查看admin接口
curl http://192.168.49.2:30005/product/list