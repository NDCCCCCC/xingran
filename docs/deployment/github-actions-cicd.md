# GitHub Actions CI/CD 手册

本文档描述仓库的两条流水线及其运维操作（配置 secrets、首次部署、手动回滚）。

## 流水线概览

| Workflow | 文件 | 触发 | 内容 |
|---|---|---|---|
| **ci** | `.github/workflows/ci.yml` | PR / push main | 后端 golangci-lint + go test；前端 lint / type-check / vitest / build |
| **deploy** | `.github/workflows/deploy.yml` | push main / push tag v* | 嵌入式构建（前端 dist → `-tags=embed` 二进制）→ scp 到服务器 → 备份/替换/重启/健康检查/回滚；tag 时另发 GitHub Release |

部署目标：腾讯云轻量服务器，`ubuntu` 用户，`/opt/xingran`，systemd 服务 `xingran`，sqlite。

## Secrets / Variables（仓库 Settings → Secrets and variables → Actions）

| 名称 | 类型 | 说明 |
|---|---|---|
| `SSH_HOST` | Secret | 服务器公网 IP |
| `SSH_PRIVATE_KEY` | Secret | 部署专用 ed25519 私钥（无 passphrase） |
| `SSH_KNOWN_HOSTS` | Secret | `ssh-keyscan -p 22 <HOST>` 的完整输出 |
| `SSH_PORT` | Variable（可选） | SSH 端口，缺省 22 |

生成与配置（本地 git-bash）：

```bash
ssh-keygen -t ed25519 -C "github-actions-xingran" -f ~/.ssh/xingran_deploy
# 公钥追加到服务器 /home/ubuntu/.ssh/authorized_keys

gh secret set SSH_HOST        --repo <owner>/xingran --body "<IP>"
gh secret set SSH_PRIVATE_KEY --repo <owner>/xingran < ~/.ssh/xingran_deploy
gh secret set SSH_KNOWN_HOSTS --repo <owner>/xingran --body "$(ssh-keyscan -p 22 <IP> 2>/dev/null)"
```

## 首次服务器准备

在服务器上以 `ubuntu` 执行仓库中的 `scripts/deploy/setup-server.sh`（幂等），再手动完成：

1. `configs/config.yaml`：从 `configs/config.prod.example.yaml` 复制并编辑（`database.type: sqlite`、端口 9000、mode release）
2. `/etc/xingran/secrets.env`：填入真实 JWT/SM4 密钥
3. 部署公钥追加到 `~/.ssh/authorized_keys`

## 发布流程

- **日常**：合并/推送到 `main` → 自动构建部署。版本号 `main-<sha7>`。
- **正式版**：`git tag v0.1.0 && git push origin v0.1.0` → 部署 + GitHub Release（附 `xingran-backend-<版本>-linux-amd64.tar.gz`，内含二进制、config 模板、systemd unit）。

验证：`curl http://<HOST>:9000/health` → `{"status":"ok","version":"<版本>",...}`

## 回滚

- **自动**：部署后健康检查失败（60s 内 `/health` 不可达或版本不符）时，流水线自动恢复最近一次备份（`/opt/xingran/xingran-backend.bak-*`，保留 5 份）并重启，随后 job 标红。
- **手动**（服务器上）：

```bash
ls -1t /opt/xingran/xingran-backend.bak-* | head   # 挑选版本
sudo install -m 0755 /opt/xingran/xingran-backend.bak-YYYYmmdd-HHMMSS /opt/xingran/xingran-backend
sudo systemctl restart xingran
curl -s http://127.0.0.1:9000/health
```

## 排障

```bash
systemctl status xingran
sudo journalctl -u xingran -n 200 --no-pager
gh run view <run-id> --log-failed          # CI 侧失败日志
```

## 已知取舍

- push main 时 CI（测试）与 deploy（部署）**并行**：build 成功即部署，不等测试结果。升级路径：将 deploy.yml 的分支触发改为 `workflow_run`（CI 成功后触发），tag 触发保持不变。
- 服务器 `config.yaml` 不由流水线管理：升级后若有新配置键，参考 Release 附件中的 `config.prod.example.yaml` 人工比对。
