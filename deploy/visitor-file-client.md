# 独立 file:// 访客页面

启用访客验证后，独立页面不可依赖跨站 Cookie。

## 旧 APK：只更新后端

旧客户端不必修改 HTML、增加 Cookie 或传 Token。Origin 缺失或为 `null` 的本地文件页面，先按旧协议向 `/visitor_login` 提交访客 ID 和名字；后端调用配置的 `/test/customer_verify` 验证，完整登录成功后才签发服务端兼容授权。之后同一来源 IP、相同 User-Agent、同一完整访客 ID 的请求可访问记录和 WebSocket。HTTP 不带 Origin 而 WS 带 `null` 的安卓旧壳会使用同一授权，两者不会因 Origin 表现不同而失配。

- 授权保存在当前 Go 进程内存，有效期固定 12 小时，不因访问自动延长，容量上限 50000 条。过期、重启、IP/浏览器标识变化时需重新打开客服完成验证登录。
- Origin 存在且非 `null`、空浏览器标识、没有成功验证登录的访问不能使用此兼容授权。
- 不同访客、不同商户不共享授权。IP 黑白名单和既有业务权限继续生效。
- 这是明确接受安全折中的旧客户端兼容模式：同公网 IP、同 User-Agent 的设备无法严格区分，知道同一访客 ID 时可能共享授权。User-Agent 不是密码。此模式不适合需要严格设备隔离的环境。
- 多个后端实例需将同一来源稳定转发到同一实例，否则应改用共享授权存储；本实现不宣称支持跨实例共享。
- 上游核验结果在 12 小时窗口内缓存，上游修改身份不会即时撤销已签发的兼容授权。

## 正式服诊断日志

`/visitor_login`、`/2/messages_page`、`/ws_visitor` 会输出 `component=visitor_diagnostic` 的结构化日志到现有日志系统（默认 `web_app.log`）。无需改 APK 或开启新的接口。WS 握手成功即时输出 `stage=upgrade, reason=connected, http_status=101`，不会等连接断开才记录。

可收集最近 100 条诊断记录：

```sh
grep '"component":"visitor_diagnostic"' web_app.log | tail -n 100
```

重点对照 `request_id`、`pid`、`visitor_hash`、`ip`、`ua_hash`、`origin`、`auth_mode`、`legacy_grant`、`reason`。`origin` 为空且 `origin_present=false` 现在是可兼容情况，不再因为缺失而拒绝；`origin_not_legacy` 表示存在一个非 `null` Origin；`not_found_for_ip_ua_id` 表示当前进程没有匹配授权（可能未登录、进程重启或 IP/UA 发生变化）；`expired` 表示已过期。记录接口的 `record_count` 仅表示返回条数，不包含正文。

诊断日志不读取请求/响应正文，不输出完整访客 ID、姓名、Cookie、Token 或 WS 子协议。Origin 和 User-Agent 有长度上限。日志仍含来源 IP 和浏览器标识，请仅交给维护人员，不公开发布。

## 可选凭证协议

新客户端可使用更强的显式凭证，避免仅凭 IP/浏览器标识识别设备：

1. POST `/visitor_login` 在原有身份参数外传 `access_mode=token`。验证成功时响应包含 `visitor_access_token`（有效期一小时）。普通同站页面继续使用 HttpOnly Cookie，不需要此参数。
2. 凭证只保存在页面内存，向同一 API 来源的请求附加 `X-Gofly-Visitor-Token`，不放入 URL、localStorage 或日志。
3. WebSocket 提供两个子协议：`gofly-visitor-v1` 和 `gofly-visitor.<凭证>`。后端核验签名、用途、访客 ID 和有效期之后，才允许 Origin 为 null 的握手。
4. HTTP 的 null Origin 仅在访客相关路径允许非 Cookie 跨域；客服后台登录接口不做此豁免。IP 黑白名单仍然生效。

凭证失效后需重新验证登录。正式环境必须使用 HTTPS/WSS，明文 HTTP/WS 会让网络中的攻击者有机会窃听访客凭证。不要记录 WebSocket 的 Sec-WebSocket-Protocol 请求头。

本地浏览器验证（仅使用自动建立、自动删除的测试数据库）：

```sh
GOFLY_TEST_MYSQL_CONFIG=/绝对路径/config/mysql.json \
GOFLY_TEST_FILE_PAGE=/绝对路径/kf/index.html \
GOFLY_TEST_PLAYWRIGHT=/绝对路径/node_modules/playwright \
go test ./router -run '^TestVerifiedVisitorHTTPAndWebSocket$' -count=1 -v
```

测试包含 file 页面验证登录、实际记录读取、无 Cookie 的 WS 心跳，以及无凭证、篡改、过期、其他访客 ID 的拒绝检查；不会请求正式服。
