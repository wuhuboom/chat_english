# 独立 file:// 访客页面

启用访客验证后，独立页面不可依赖跨站 Cookie。后端和前端必须一起更新。

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
