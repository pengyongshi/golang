# 功能介绍
监控哨兵状态，每隔一定时间内抓取redis master IP。如果检查与上一次记录的masterIP不一致。
则重启namespace下所有的deployment服务。

# 参数介绍
|参数|默认值|是否必须|描述|
|----|----|----|----|
|--kubeconfig|~/.kube/config|*|指定kubeconfig文件，一般是/etc/kubernetes/admin.comf|
|--k8s-master-url| | * |指定kube-apiserver的地址，用于连接到k8s集群。如：https://192.168.10.51:6443|
|--sentinel-address| | * | sentinel地址连接池。以,分割。如: 192.168.10.1:26379,192.168.10.2:26379|
|--namespace|entuc,entcmd| |指定要重启的命名空间，以,分割。如: entuc,entcmd|
|--sentinel-name|mymaster| | 哨兵集群名称|
|--sentinel-status|status.json| | 记录节点状态的json文件|
|--interval-time|30| | 探测的间隔时长，单位为s|

