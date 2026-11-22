// Invoked by the isolated Go integration test, never against a production API.
const { chromium } = require(process.env.GOFLY_TEST_PLAYWRIGHT);
const { pathToFileURL } = require('node:url');
const assert = require('node:assert/strict');
(async () => {
  const origin = process.env.GOFLY_TEST_APP_URL;
  assert.match(origin, /^http:\/\/127\.0\.0\.1:\d+$/);
  const browser = await chromium.launch({ channel: 'chrome', headless: true });
  try {
    const context = await browser.newContext();
    const page = await context.newPage();
    page.on('pageerror', error => console.log('PAGE ERROR:', error.message));
    page.on('websocket', ws => ws.on('socketerror', error => console.log('WS ERROR:', error)));
    await page.route('**/*', route => {
      const url = new URL(route.request().url());
      if (url.origin === origin && url.pathname === '/') return route.fulfill({body:''});
      return url.protocol === 'file:' || url.origin === origin ? route.continue() : route.abort();
    });
    const url = pathToFileURL(process.env.GOFLY_TEST_FILE_PAGE);
    url.search = new URLSearchParams({ api: origin, ent_id: '1', kefu_id: 'visitor-handler', visitor_id: 'fixture_id', visitor_name: '张三 & Co', extra:'e30=' }).toString();
    const history = page.waitForResponse(r => new URL(r.url()).pathname === '/2/messages_page' && r.status() === 200);
    const pong = new Promise(resolve => page.on('websocket', ws => ws.on('framereceived', frame => {
      try { if (JSON.parse(frame.payload).type === 'pong') resolve(); } catch (_) {}
    })));
    await page.goto(url.href);
    const res = await history;
    const data = await res.json();
    assert(data.result.list.some(row => row.content === 'test message'));
    await page.waitForFunction(() => document.querySelector('#app')?.__vue__?.socket?.readyState === 1);
    await page.evaluate(() => document.querySelector('#app').__vue__.socket.send(JSON.stringify({type:'ping'})));
    await Promise.race([pong, new Promise((_, reject) => setTimeout(() => reject(Error('No WebSocket pong')), 5000))]);
    assert.equal((await context.cookies(origin)).length, 0);
    const other = await browser.newContext();
    const anonymous = await other.request.get(origin + '/2/messages_page?visitor_id=1%7Cfixture_id&ent_id=1&page=1&pagesize=5');
    assert.equal(anonymous.status(), 404);
    await other.close();
    console.log('PASS: real file page login, persisted history, WebSocket pong without cookies; separate anonymous client denied');
  } finally { await browser.close(); }
})().catch(error => { console.error(error); process.exitCode = 1; });
