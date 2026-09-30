const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
function load(source, lang) {
    let options;
    const requests = [], messages = [];
    const context = {LANG:lang, KEFU_ID:'test', window:{}, getWsBaseUrl:()=>'',
        GOFLY_LANG:{en:{},cn:{},'en-US':{}}, console:{log(){}},
        Vue:function(value){ options=value; }, $:{ajax:r=>requests.push(r)}};
    vm.createContext(context);
    vm.runInContext(fs.readFileSync("static/js/chat-lang.js","utf8"),context);
    context.GOFLY_LANG[lang] = context.GOFLY_LANG[lang] || {};
    vm.runInContext(source,context);
    const instance = Object.assign({},options.data,options.methods,{$message:m=>messages.push(m.message)});
    return {instance,requests,messages};
}
const source=fs.readFileSync('static/js/chat-page.js','utf8');
const error={code:409,msg:'访客连接已变更，请重新连接'};
for(const lang of ['en','en-US','cn']){
    const t=load(source,lang);let acknowledged=0;
    const send=()=>t.instance.sendAjax('/2/messages_read','post',{},()=>acknowledged++);
    send();assert.equal(t.requests.length,0,'disconnected read must not submit');
    t.instance.wsSocketClosed=false;send();send();assert.equal(t.requests.length,1,'single in-flight request');
    t.requests[0].success(error);t.requests[0].complete();
    assert.equal(t.messages[0],lang==='cn'?error.msg:'Your chat connection has changed. Please reconnect.');
    assert.equal(acknowledged,0,'failed receipt must not mark messages read');
    send();assert.equal(t.requests.length,1,'retry backoff');
    t.instance.readReceiptRetryAt=0;send();t.requests[1].success({code:200,msg_ids:[42]});t.requests[1].complete();
    assert.equal(acknowledged,1);
    t.instance.sendAjax('/2/message_ask','post',{},()=>{});
    t.requests[2].error({responseText:'<html>Bad gateway</html>'});
    assert.ok(t.messages[1]);
    if(lang!=='cn')assert.ok(!/[\u3400-\u9fff]/.test(t.messages[1]));
}
console.log('PASS: English/Chinese errors, disconnected gate, in-flight deduplication, backoff, recovery, non-JSON errors');
