# 编译并运行

1. `go get -u` 安装依赖
2. `go build .` 编译生成 `neko-relay`
3. `./neko-relay -c config.yaml` 运行

也可以直接下载release已编译好的文件直接运行

`config.yaml`说明详见文件注释

# API列表

|路径|操作名称|POST BODY|
|-|-|-|
|`/add`|添加规则|`{rid,remote,rport,type}`|
|`/edit`|编辑规则|`{rid,remote,rport,type}`|
|`/del`|删除规则|`{rid}`|
|`/sync`|添加规则|`[{rid,remote,rport,type}, ...]`|
|`/traffic`|流量统计||
|`/stat`|服务器状态||

---

Powered by [Neko Neko Relay](https://relay.nekoneko.cloud)