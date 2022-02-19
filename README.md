# ksm-checker
Check ksm metrics and runing with cronjob,get the metrics from KSM and apiserver, then we compare they,create a alert when KSM have some data of not exist in apiserver. see issue: https://github.com/kubernetes/kube-state-metrics/issues/1679

这个仓库是为了检查KSM的metrics中是否有一些apiserver已经不存在(删除了资源)的数据,这个程序拉取KSM的metrics数据并且解析出pod信息,并且通过LIST API从apiserver查询,对比得出KSM是否有一些过时的数据.具体的issue可以看这里:https://github.com/kubernetes/kube-state-metrics/issues/1679

