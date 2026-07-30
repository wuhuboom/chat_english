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
