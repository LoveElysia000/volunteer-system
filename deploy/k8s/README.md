# deploy/k8s — Kubernetes (k3s) 部署清单

## 前提

- 服务器已安装 k3s（`curl -sfL https://rancher-mirror.rancher.cn/k3s/k3s-install.sh | INSTALL_K3S_MIRROR=cn INSTALL_K3S_VERSION=v1.36.4+k3s1 sh -`）
- GitHub 仓库已配置 `KUBE_CONFIG` secret（k3s 的 kubeconfig 全文，server 地址为公网 IP）
- 安全组/防火墙放行：**6443**（CI 部署）、**30109**（应用）、**30080**（Headlamp 界面）

## 文件说明

| 文件 | 内容 |
|---|---|
| `00-namespace.yaml` | 命名空间 volunteer-system |
| `01-config.yaml` | 应用配置模板（ConfigMap）+ MySQL 建表脚本（ConfigMap） |
| `02-mysql.yaml` | MySQL 8 StatefulSet + Service（10Gi 数据卷，首次启动自动执行 ddl.sql） |
| `03-redis.yaml` | Redis 7 StatefulSet + Service（2Gi 数据卷） |
| `04-app.yaml` | 应用 Deployment（2 副本滚动更新）+ uploads PVC + NodePort Service(30109) |
| `kustomization.yaml` | 一键 apply 全部：`kubectl apply -k deploy/k8s/` |
| `../headlamp.yaml` | Headlamp 可视化界面（独立于应用，NodePort 30080） |

## 部署方式（GitOps：服务器上无需克隆仓库）

清单由 GitHub Actions（`.github/workflows/cd.yml` 的 deploy job）自动 apply，
触发方式：**push 到 main**，或在 **Actions 页面手动 Run workflow**。

服务器上唯一需要手动做的：创建两个集群内 Secret（真实密码不进 git）。

## 首次部署步骤

### 1. 触发一次 CD（GitHub 网页上点 Actions → CD → Run workflow）

这会构建镜像并 apply 全部清单（含命名空间和 Headlamp）。
此时应用 Pod 起不来是正常的——Secret 还没创建，CD 日志会给出黄色 warning 提示。

### 2. 在服务器上创建 Secret

> 需要各密钥的真实值（与 `.env` 一致）。若 `.env` 在本机：`scp .env root@<服务器IP>:~/`

```bash
set -a; source ~/.env; set +a

# 应用密钥
kubectl -n volunteer-system create secret generic volunteer-secrets \
  --from-literal=APP_SECRET_KEY="$APP_SECRET_KEY" \
  --from-literal=JWT_SECRET="$JWT_SECRET" \
  --from-literal=MYSQL_DATABASE="$MYSQL_DATABASE" \
  --from-literal=MYSQL_USER="$MYSQL_USER" \
  --from-literal=MYSQL_PASSWORD="$MYSQL_PASSWORD" \
  --from-literal=MYSQL_ROOT_PASSWORD="$MYSQL_ROOT_PASSWORD" \
  --from-literal=AI_API_KEY="${AI_API_KEY:-}"

# GHCR 镜像拉取凭据（PAT 生成：github.com/settings/tokens → classic → read:packages）
kubectl -n volunteer-system create secret docker-registry ghcr-creds \
  --docker-server=ghcr.io \
  --docker-username='<GitHub 用户名>' \
  --docker-password='<PAT>'
```

Secret 创建后，起不来的 Pod 会自动恢复（也可再去 Actions 跑一次 CD 加速收敛）。

### 3. 验证

```bash
# 服务器上（或任何能访问 30109 的地方）
curl http://<服务器IP>:30109/healthz     # {"checks":{"mysql":"ok","redis":"ok"},...}

# 浏览器：Headlamp 可视化界面
# http://<服务器IP>:30080  → Token 登录
# 登录 token：kubectl -n headlamp create token headlamp-admin
```

## 日常操作

```bash
# 更新应用：push 代码即可（CI 自动构建镜像 + 滚动更新）
# 手动改镜像（在服务器或任何有 kubeconfig 的机器上）：
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
