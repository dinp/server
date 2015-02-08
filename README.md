Server
==========

这是整个系统最核心的模块了，主要有以下功能

- 1. 开一个rpc端口，接收Agent的心跳信息，收集所有计算节点的负载和container列表，存入内存，称之为real state
- 2. 开一个http端口用于调试，暴露内存信息
- 3. 连接Dashboard的DB，定期获取当前app的目标状态，称之为desired state
- 4. 比较real state和desired state，如果container挂了，或者扩容了，就可以比较得知差异，然后创建新的container或者销毁多余的container
- 5. 分析Agent上报的container列表，组织出路由信息，写入redis
- 6. 把配置的scribe连接地址通过环境变量写入container，container中的app就可以把log推到这个scribe服务器

## 问题

**如果server挂了怎么办？**

Agent配置了两个server的地址，平时只启动一个server，如果server所在机器挂了，去启动另一个即可，Agent会自动重连。此处可以做一个自动选主之类的逻辑，太麻烦，没做

**如果存放路由信息的redis挂了怎么办？**

router每次从redis获取了路由之后会缓存到本地内存，redis挂了问题不大，只是不能感知后端container变化了。

*其实，即使dinp的大部分组件都挂了，只要router没挂，container没挂，已有的服务都不会受影响，只是没法上线、扩容了*
