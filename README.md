一个分布式缓存框架（仿照 groupcache 实现，当然现在看起来已经没什么关系了）
### 主要特点
* 代码小巧，适合作为学习向的项目
* 易拓展。注册中心、缓存淘汰策略、RPC 协议、数据分片策略均可轻易替换实现
### 快速开始
```go
var (
    dbHeight = map[string]string{
        "laowang":  "173", 
        "xiaoming": "166",
    }
    dbScore = map[string]string{
        "laowang":  "96", 
        "xiaoming": "97",
    }
)

func main() {
    c := gcache.New()
    c.Group("height", 1<<10, gcache.EvictionLru, time.Minute, gcache.ShardingConsistenthash,
        gcache.DBGetterFunc(func(key string) ([]byte, error) {
            return []byte(dbHeight[key]), nil
        }))
    c.Group("score", 1<<10, gcache.EvictionLru, time.Minute, gcache.ShardingConsistenthash, 
        gcache.DBGetterFunc(func(key string) ([]byte, error) {
            return []byte(dbScore[key]), nil
        }))
    c.Registry(gcache.RegistryZK, "xxxxxxxxx")
    c.Protocol(gcache.ProtocolHTTP)
    err := c.Start(":6060")
    if err != nil {
        log.Fatal("server start fail: " + err.Error())
    }
}
```
