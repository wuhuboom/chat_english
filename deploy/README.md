# GOFLY production deployment

This deployment keeps the Go process in the foreground under systemd and lets
systemd own restarts. Do not add the application's `-d` flag to `ExecStart`;
double-daemonizing prevents systemd from tracking the real process reliably.

## 1. Install the application

```bash
go build -o main .
sudo install -d -o gofly -g gofly /opt/gofly
sudo cp -a main static config /opt/gofly/
sudo chown -R gofly:gofly /opt/gofly
```

Create the service account first when it does not exist:

```bash
sudo useradd --system --home /opt/gofly --shell /usr/sbin/nologin gofly
```

## 2. Install the process supervisor

```bash
sudo install -d -m 0750 /etc/gofly
sudo cp deploy/gofly.env.example /etc/gofly/gofly.env
sudo cp deploy/gofly.service /etc/systemd/system/gofly.service
sudo systemctl daemon-reload
sudo systemctl enable --now gofly
sudo systemctl status gofly
```

The service restarts after crashes with a three-second delay and raises the
open-file limit for concurrent WebSocket sessions.

When HTTPS terminates at nginx, set `GOFLY_BROWSER_ORIGIN` in
`/etc/gofly/gofly.env` to the exact public origin (for example,
`https://chat.example.com`, without a trailing slash). Preserve the public
Host header, including a non-default port if applicable. Staff browser sessions
use a host-only HttpOnly cookie with SameSite=Strict. HTTP login does not
require extra configuration; passwords and session cookies on HTTP travel
without transport encryption. HTTPS cookies remain Secure.
Logging out revokes all sessions for that staff account.

## 3. Install the reverse proxy

Copy `deploy/nginx-gofly.conf` to the server's nginx `conf.d` directory, replace
`chat.example.com`, then attach the site's existing TLS certificate if HTTPS is
used.

```bash
sudo nginx -t
sudo systemctl reload nginx
```

The WebSocket proxy read/send timeouts are 300 seconds. They must remain longer
than the application's 75-second protocol Pong deadline.

## 4. Verify

```bash
curl -fsS http://127.0.0.1:8081/healthz
curl -fsS http://127.0.0.1:8081/readyz
sudo journalctl -u gofly -f
sudo systemctl is-active gofly nginx
```

WebSocket logs now include `role`, account/visitor `id`, `session`, close
`code`, close `reason`, and the number of remaining active sessions:

- `1000`: normal close.
- `1001`: page navigation, browser suspension, or a normal client departure.
- `1006`: abnormal network/proxy termination; correlate its timestamp with
  nginx and load-balancer logs.
- `code=0`: a lower-level transport error such as reset or broken pipe.

When one browser tab closes, `active` must stay above zero while another tab for
the same agent is connected. Automatic visitor failover starts only after the
last agent session disappears and the configured grace period expires.

After pulling an update, rebuild the binary and restart the service before
checking /readyz. Git updates the checkout, but the running process keeps using
the old binary until it is restarted.

## 关闭普通访客

在客服服务的 `config.yaml` 填写业务后台的完整校验地址。

```yaml
app:
  customer_verify_url: "https://business.example.com/test/customer_verify"
```

客服服务会发起 GET 请求，将来访链接的 `visitor_id` 作为 `code`，
将 `visitor_name` 作为 `username`。业务后台应返回 HTTP 200，响应体为
JSON 布尔值 `true` 或 `false`。缺少参数、返回 `false`、超时或校验服务报错时，
访客请求返回 HTTP 404。

用超级管理员打开「系统管理 → 配置参数 → 访客访问」，启用「关闭普通访客」
并保存。开关默认关闭。访客在新浏览器打开原链接时会重新校验，校验成功后
可访问原会话。若页面嵌入不同站点且浏览器拦截第三方 Cookie，请用新标签页
打开访客链接。

## 访客 IP 白名单

在「IP 黑名单」页启用 IP 白名单。单个规则支持多个 IPv4、IPv6 地址和地区，
匹配其中一项即可通过。超级管理员保存全局规则，商户只管理自己的规则。
同时启用全局和商户规则时，访客须通过两边的规则。已有 IP 黑名单继续拦截。

反向代理部署时，应在 `app.trusted_proxies` 列出客服服务到访客之间需要信任的
代理地址，并转发 `X-Forwarded-For`。地区匹配使用
`config/GeoLite2-Country.mmdb`。无法识别地区的地址可通过精确 IP 条目放行。
