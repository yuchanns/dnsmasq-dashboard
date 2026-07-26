# Leaseboard

Leaseboard 是一个面向 dnsmasq 的只读 DHCP 租约看板。它将 dnsmasq 租约文件与 Linux 邻居表合并，提供设备在线状态、租约剩余时间、地址池占用率和 MAC 冲突提示。

## 特性

- 只读数据路径，不修改 dnsmasq 配置或租约
- 通过 Server-Sent Events 实时推送有效状态变化
- 合并 `ip -j neighbor`，区分在线、最近出现、离线和冲突
- 支持设备、IP、MAC 搜索，状态过滤和多字段排序
- 单个 Go 二进制内嵌前端
- 提供 `linux/amd64` 和 `linux/arm64` 容器镜像

## Docker 部署

使用 host 网络才能读取宿主机邻居表。租约目录应只读挂载，避免 dnsmasq 原子替换租约文件时容器保留旧 inode。

```yaml
services:
  leaseboard:
    image: ghcr.io/yuchanns/dnsmasq-dashboard:latest
    network_mode: host
    restart: unless-stopped
    environment:
      LISTEN_ADDRESS: 0.0.0.0:8080
      LEASE_FILE: /host-dnsmasq/dhcp.leases
      NETWORK_INTERFACE: eth0
      NETWORK_NAME: Local network
      POOL_START: 192.168.1.100
      POOL_END: 192.168.1.249
    volumes:
      - /var/lib/misc:/host-dnsmasq:ro
    read_only: true
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
```

## 配置

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `LISTEN_ADDRESS` | `0.0.0.0:8080` | HTTP 监听地址 |
| `LEASE_FILE` | `/var/lib/misc/dnsmasq.leases` | dnsmasq 租约文件 |
| `NETWORK_INTERFACE` | `eth0` | 邻居表网络接口 |
| `IP_COMMAND` | `ip` | `iproute2` 命令路径 |
| `NETWORK_NAME` | `Local network` | 页面展示的网络名称 |
| `POOL_START` | `192.168.1.100` | 地址池起始 IPv4 |
| `POOL_END` | `192.168.1.249` | 地址池结束 IPv4 |
| `POLL_INTERVAL_SECONDS` | `2` | 数据采样间隔，范围 1–60 秒 |

`NEIGHBOR_FILE` 仅用于开发和测试。设置后将从 JSON 文件读取邻居数据，不执行 `ip` 命令。

## 本地开发

```bash
cd web
npm ci
npm run build
cd ..
go test ./...
go run ./cmd/leaseboard
```

## 安全边界

Leaseboard 不提供租约释放、静态绑定、dnsmasq 重载或任意文件读取接口。生产部署建议保留只读根文件系统、只读租约目录、`cap_drop: ALL` 和 `no-new-privileges`。

## License

[MIT](LICENSE)
