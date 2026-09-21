# deploy/k8s — Kubernetes (k3s) 部署清单

## 前提

- 服务器已安装 k3s（`curl -sfL https://rancher-mirror.rancher.cn/k3s/k3s-install.sh | INSTALL_K3S_MIRROR=cn sh -`）
- 本机 kubeconfig 已指向该集群（`KUBECONFIG=~/.kube/volunteer-config`）
- 阶段 1 的健康探针代码（`/livez`、`/healthz`）已推送，CI 已构建出最新镜像

## 文件说明

| 文件 | 内容 |
|---|---|
| `00-namespace.yaml` | 命名空间 volunteer-system |
| `01-config.yaml` | 应用配置模板（ConfigMap）+ MySQL 建表脚本（ConfigMap） |
| `02-mysql.yaml` | MySQL 8 StatefulSet + Service（10Gi 数据卷，首次启动自动执行 ddl.sql） |
| `03-redis.yaml` | Redis 7 StatefulSet + Service（2Gi 数据卷） |
| `04-app.yaml` | 应用 Deployment（2 副本滚动更新）+ uploads PVC + NodePort Service(30109) |

## 首次部署步骤

### 1. 创建 Secret（敏感信息不入库）

```bash
export KUBECONFIG=~/.kube/volunteer-config

# 应用密钥（值从 .env 取）
kubectl -n volunteer-system create secret generic volunteer-secrets \
  --from-literal=APP_SECRET_KEY='<值>' \
  --from-literal=JWT_SECRET='<值>' \
  --from-literal=MYSQL_DATABASE='volunteer_system' \
  --from-literal=MYSQL_USER='volunteer' \
  --from-literal=MYSQL_PASSWORD='<值>' \
  --from-literal=MYSQL_ROOT_PASSWORD='<值>' \
  --from-literal=AI_API_KEY=''

# GHCR 镜像拉取凭据（PAT 需有 read:packages 权限）
kubectl -n volunteer-system create secret docker-registry ghcr-creds \
  --docker-server=ghcr.io \
  --docker-username='<GitHub 用户名>' \
  --docker-password='<PAT>'
```

### 2. 部署（一条命令）

```bash
kubectl apply -k deploy/k8s/
```

等效于按顺序 apply 目录下全部清单（00-namespace → 01-config → 02-mysql → 03-redis → 04-app）。

### 3. 验证

```bash
kubectl -n volunteer-system get pods -w          # 全部 Running/Ready
curl http://<服务器IP>:30109/healthz             # {"checks":{"mysql":"ok","redis":"ok"},...}
```

## 日常操作

```bash
# 更新镜像（CD 改造后的动作）
kubectl -n volunteer-system set image deployment/volunteer-app \
  app=ghcr.io/loveelysia000/volunteer-system:<新tag>

# 查看日志（stdout 收集）
kubectl -n volunteer-system logs -f deploy/volunteer-app

# 重置数据库（危险：删除 MySQL 数据卷后重新初始化）
# kubectl -n volunteer-system delete statefulset mysql --cascade=orphan
# kubectl -n volunteer-system delete pvc data-mysql-0
```

## 已知事项

- **MySQL 建表脚本只在数据卷为空时执行**；变更表结构需手动 `kubectl exec` 进 MySQL 执行增量 SQL
- **镜像拉取失败**（国内网络拉 docker.io/mysql/redis 超时）时，给 k3s 配置镜像加速：
  写 `/etc/rancher/k3s/registries.yaml` 配 mirror 后 `sudo systemctl restart k3s`
- 2 个应用副本共享 uploads PVC（ReadWriteMany，k3s local-path 支持）；若未来换 RWO 存储类需回到单副本或改对象存储
- 数据库备份未包含在本清单内，需要时用 `kubectl exec` 跑 mysqldump 定时任务
