## Shua!
支持多线程，自定义休息时间的一个小工具，用于刷流量，亦可用于PCDN刷下行。

## 下载
### Github Release
从 [Github Release](https://github.com/ZeroWolf233/shua/releases) 下载最新版。
### Docker
```bash
docker run -d \
  --name shua \
  --restart unless-stopped \
  -e u=https://adventurex-public.s3.ap-east-1.amazonaws.com/adventureX-logo.svg \
  -e w=256 \
  -e i=0s \
  zerowolf233/shua:latest
```

## 使用
### 可选flag
| 名称    | 默认值                                                                                                             | 说明            |
|-------|-----------------------------------------------------------------------------------------------------------------|---------------|
| url   | https://js.a.kspkg.com/kos/nlav10814/kwai-android-generic-gifmakerrelease-13.7.30.43728_x64_5d82bf.apk          | 请求内容的地址       |
| w     | 64                                                                                                              | 创建多少个工作进程(线程) |
| i     | 0s                                                                                                              | 每次请求后的休息时长    |
| ua    | Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36 | 自定义的ua请求头     |
| address | 无                                                                                                               | 源IP地址         |
| 4     | 否                                                                                                               | 仅IPv4         |
| 6     | 否                                                                                                               | 仅IPv6         |
| rate  | 无                                                                                                               | 限速 (如23.33m)  |

### 使用示例
```bash
./shua -url https://lf5-j1gamecdn-cn.dailygn.com/obj/lf-game-lf/gdl_app_2682/1233880772355.mp4 -w 128 -i 0s -6 -rate 23.33m
```
