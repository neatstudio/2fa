# 2FA CLI

一个本地命令行 TOTP/2FA 验证码管理工具，用来替代手机 authenticator app 的日常查看场景。

运行 `2fa list` 后，它会在终端中显示所有已绑定账号的当前验证码、倒计时、分组和备注；也可以按分组过滤，例如 `2fa list game`、`2fa ls work`。

## 功能

- 本地保存 TOTP 账号和 base32 secret
- 生成标准 6 位 TOTP 验证码
- 默认 30 秒周期，HMAC-SHA1，兼容常见 authenticator app
- 表格展示分组、账号、验证码、剩余秒数、备注
- 交互终端中默认每秒刷新
- 管道、重定向或 `--once` 模式下单次输出
- 支持分组过滤
- 支持新增、编辑、删除账号
- 默认数据文件权限限制为当前用户读写

## 安装

本项目是 Go CLI。

```sh
go test ./...
go build -o bin/2fa ./cmd/2fa
```

如果 `~/.local/bin` 在你的 `PATH` 中，可以创建软链接：

```sh
ln -s "$PWD/bin/2fa" ~/.local/bin/2fa
```

也可以直接安装到 Go bin 目录：

```sh
go install ./cmd/2fa
```

确保 `$(go env GOPATH)/bin` 在 `PATH` 中。

## 使用

查看帮助和版本：

```sh
2fa --help
2fa --version
```

新增账号：

```sh
2fa add --name github --secret "JBSWY3DPEHPK3PXP" --group work --note "GitHub admin login"
```

查看帮助：

```sh
2fa
```

查看所有账号：

```sh
2fa list
2fa ls
```

只输出一次，适合脚本或复制：

```sh
2fa list --once
```

按分组查看：

```sh
2fa list game
2fa ls work --once
```

启动本地 Web UI：

```sh
2fa serve
```

启动后终端会打印带进程内临时会话 token 的访问地址，例如：

```text
2fa web UI: http://127.0.0.1:23832/?token=...
```

默认会监听：

- `127.0.0.1:23832`
- 当前机器上属于 `100.64.0.0/10` 的地址，例如 Tailscale/CGNAT 地址

默认不会监听 `0.0.0.0` 或公网网卡。这里的 `100` 网段按 Tailscale/CGNAT 的 `100.64.0.0/10` 处理，不是整个 `100.0.0.0/8`；如果确实需要其它网段，可以显式放行：

```sh
2fa serve --port 23833
2fa serve --addr 100.101.102.103:23832
2fa serve --addr 0.0.0.0:23832 --allow 192.168.1.0/24
2fa serve --allow 100.0.0.0/8
```

Web UI 支持：

- 实时查看验证码和倒计时
- 按分组过滤
- 新增账号
- 编辑 group/note/secret
- 删除账号

Web UI 不会在 API 或页面中返回原始 secret；编辑时 secret 输入框不会预填，留空表示不修改。

编辑账号：

```sh
2fa edit github --group account
2fa edit github --note "primary admin account"
2fa edit github --secret "JBSWY3DPEHPK3PXP"
```

删除账号：

```sh
2fa delete github
```

跳过删除确认：

```sh
2fa delete github --yes
```

## 数据存储

默认数据文件：

```text
~/.2fa/accounts.json
```

默认行为：

- `~/.2fa` 目录权限为 `0700`
- `~/.2fa/accounts.json` 文件权限为 `0600`
- 普通列表输出不会显示原始 secret
- 写入时使用同目录临时文件和 rename，降低写坏文件的概率

开发或测试时可指定自定义存储文件：

```sh
2fa --store /tmp/accounts.json --once
```

如果自定义 store 的父目录已经存在，程序不会修改该父目录权限，但仍会确保账号文件为 `0600`。

## 备份和恢复

备份：

```sh
cp ~/.2fa/accounts.json ~/accounts.json.backup
chmod 600 ~/accounts.json.backup
```

恢复：

```sh
mkdir -p ~/.2fa
chmod 700 ~/.2fa
cp ~/accounts.json.backup ~/.2fa/accounts.json
chmod 600 ~/.2fa/accounts.json
```

## 安全边界

本工具把 TOTP secret 以明文 base32 保存到本地 JSON 文件中，只依赖本地文件权限保护。

`2fa serve` 会生成进程内临时 token，并对请求来源做默认限制：允许 `127.0.0.0/8`、`::1/128` 和 `100.64.0.0/10`。即便如此，Web 页面里的实时验证码也属于敏感信息，不要把服务暴露给不可信网络。

它不会：

- 加密 secret
- 使用系统 Keychain
- 防止同一用户权限下的恶意进程读取文件
- 防止机器被入侵后的 secret 泄露

请把 `~/.2fa/accounts.json` 和它的备份都当作敏感文件处理。

## 时间同步

TOTP 依赖本机系统时间。如果验证码被服务端拒绝，请先确认系统时间和时区正确，并开启系统时间同步。

## 发布打包

本项目通过 GitHub Actions 自动发布。

触发方式：

- 推送 `v*` tag，例如 `v1.0.0`
- 或在 GitHub Actions 页面手动运行 `Release` workflow，并填写版本号

Release workflow 会：

- 运行 `go test ./...`
- 交叉编译 macOS/Linux/Windows 的 amd64/arm64 产物
- 生成 `SHA256SUMS`
- 生成 `RELEASE_NOTES.md`
- 创建 GitHub Release 并上传归档文件

本地也可以复用同一套脚本：

```sh
make package VERSION=v1.0.0
```

产物会生成到 `dist/`。

## 开发

运行测试：

```sh
go test ./...
```

临时 store 冒烟测试：

```sh
store=$(mktemp)
2fa --store "$store" add --name demo --secret JBSWY3DPEHPK3PXP --group test --note demo
2fa --store "$store" list test --once
2fa --store "$store" delete demo --yes
```
