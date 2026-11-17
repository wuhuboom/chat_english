if (window.jQuery && document.currentScript && document.currentScript.hasAttribute("data-staff-requests")) {
    window.jQuery.ajaxPrefilter(function (options, originalOptions, request) {
        if (options.crossDomain === false) {
            request.setRequestHeader("X-Gofly-Login", "1");
        }
    });
}

function getBaseUrl() {
    var ishttps = 'https:' == document.location.protocol ? true : false;
    var url = window.location.host;
    if (ishttps) {
        url = 'https://' + url;
    } else {
        url = 'http://' + url;
    }
    return url;
}
function getWsBaseUrl() {
    var ishttps = 'https:' == document.location.protocol ? true : false;
    var url = window.location.host;
    if (ishttps) {
        url = 'wss://' + url;
    } else {
        url = 'ws://' + url;
    }
    return url;
}
function notify(title, options, callback) {
    if (!window.Notification) {
        console.log("浏览器不支持notify");
        return Promise.resolve(false);
    }

    function showNotification() {
        var notification = new Notification(title, options);
        if (callback) {
            notification.onclick = function(event) {
                callback(notification, event);
            }
        }
        setTimeout(function () {
            notification.close();
        },8000);
        return true;
    }

    if (Notification.permission === 'granted') {
        return Promise.resolve(showNotification());
    }
    if (Notification.permission === 'denied') {
        return Promise.resolve(false);
    }
    return Notification.requestPermission().then(function(permission) {
        return permission === 'granted' ? showNotification() : false;
    }).catch(function() {
        return false;
    });
}
var titleTimer=0;
var titleNum=0;
var originTitle = document.title;
function flashTitle() {
    if(titleTimer!=0){
        return;
    }
    titleTimer = setInterval(function(){
        titleNum++;
        if (titleNum == 3) {
            titleNum = 1;
        }
        if (titleNum == 1) {
            document.title = '【☆】' + originTitle;
        }
        if (titleNum == 2) {
            document.title = '【★】' + originTitle;
        }
    }, 500);

}
function clearFlashTitle() {
    clearInterval(titleTimer);
    document.title = originTitle;
}
var faceTitles = ["[a]", "[b]", "[c]", "[d]", "[e]", "[f]", "[g]", "[h]", "[i]", "[j]", "[k]", "[l]", "[m]", "[n]", "[o]", "[p]", "[q]", "[r]", "[s]", "[t]", "[u]", "[v]", "[w]", "[x]", "[y]", "[z]", "[aa]", "[bb]", "[cc]", "[dd]", "[ee]", "[ff]", "[gg]", "[hh]", "[ii]", "[jj]", "[kk]", "[ll]", "[mm]", "[nn]", "[oo]", "[pp]", "[qq]", "[rr]", "[ss]", "[tt]", "[uu]", "[vv]", "[ww]", "[xx]", "[yy]", "[zz]", "[a1]", "[b1]", "[good]", "[NO]", "[c1]", "[d1]", "[e1]", "[f1]", "[g1]", "[h1]", "[i1]", "[g1]", "[k1]", "[l1]", "[m1]", "[n1]", "[o1]", "[p1]", "[q1]", "[cake]"];
function placeFace() {
    var faces=[];
    for(var i=0;i<faceTitles.length;i++){
        faces[faceTitles[i]]="/static/images/face/"+i+".gif";
    }
    return faces;
}

/**
 * H5 聊天使用的现代 Unicode Emoji 集合。
 * 保留 placeFace() 供历史 face[a] 消息继续渲染，新增消息直接发送 Unicode 字符。
 */
function isEnglishLanguage(language) {
    return String(language || "").toLowerCase().split(/[-_]/)[0] === "en";
}

function getEmojiPickerTexts(language) {
    if (isEnglishLanguage(language)) {
        return {
            insert: "Insert emoji",
            categories: "Emoji categories",
            recent: "Recently used",
            fallback: "Emoji"
        };
    }
    return {
        insert: "插入表情",
        categories: "表情分类",
        recent: "最近使用",
        fallback: "表情"
    };
}

function getModernEmojiGroups(language) {
    var english = isEnglishLanguage(language);
    var groupName = function (chinese, englishText) {
        return english ? englishText : chinese;
    };
    return [
        {
            id: "smileys",
            icon: "😀",
            name: groupName("表情与情绪", "Smileys & Emotion"),
            emojis: [
                "😀","😃","😄","😁","😆","😅","😂","🤣","🥲","😊","😇","🙂",
                "🙃","😉","😌","😍","🥰","😘","😗","😙","😚","😋","😛","😝",
                "😜","🤪","🤨","🧐","🤓","😎","🥸","🤩","🥳","🙂‍↕️","😏","😒",
                "😞","😔","😟","😕","🙁","☹️","😣","😖","😫","😩","🥺","😢",
                "😭","😤","😠","😡","🤬","🤯","😳","🥵","🥶","😶‍🌫️","😱","😨",
                "😰","😥","😓","🤗","🤔","🫣","🤭","🫢","🫡","🤫","🫠","🤥",
                "😶","🫥","😐","🫤","😑","😬","🙄","😯","😦","😧","😮","😲",
                "🥱","😴","🤤","😪","😵","😵‍💫","🫨","🤐","🥴","🤢","🤮","🤧",
                "😷","🤒","🤕","😈","👿","💩","🤡","👻","💀","☠️","👽","🤖"
            ]
        },
        {
            id: "gestures",
            icon: "👋",
            name: groupName("人物与手势", "People & Gestures"),
            emojis: [
                "👋","🤚","🖐️","✋","🖖","🫱","🫲","🫳","🫴","👌","🤌","🤏",
                "✌️","🤞","🫰","🤟","🤘","🤙","👈","👉","👆","👇","☝️","🫵",
                "👍","👎","✊","👊","🤛","🤜","👏","🙌","🫶","👐","🤲","🤝",
                "🙏","✍️","💅","🤳","💪","🦾","🧠","👀","👁️","👄","🫦","👶",
                "🧒","👦","👧","🧑","👨","👩","🧓","👴","👵","🙋","🙆","🙅",
                "🤷","🤦","🧏","🙇","💁","🧘","🏃","🚶","🧍","🕺","💃","👯"
            ]
        },
        {
            id: "hearts",
            icon: "❤️",
            name: groupName("爱心与符号", "Hearts & Symbols"),
            emojis: [
                "❤️","🩷","🧡","💛","💚","💙","🩵","💜","🤎","🖤","🩶","🤍",
                "💔","❤️‍🔥","❤️‍🩹","❣️","💕","💞","💓","💗","💖","💘","💝","💟",
                "💋","💯","💢","💥","💫","💦","💨","🕳️","💬","👁️‍🗨️","🗨️","🗯️",
                "💭","💤","✨","⭐","🌟","🔥","🎉","🎊","✅","❌","⭕","❗",
                "❓","‼️","⁉️","⚠️","🚫","♻️","🔞","🆗","🆕","🆒","🆘","🆙"
            ]
        },
        {
            id: "animals",
            icon: "🐶",
            name: groupName("动物与自然", "Animals & Nature"),
            emojis: [
                "🐶","🐱","🐭","🐹","🐰","🦊","🐻","🐼","🐻‍❄️","🐨","🐯","🦁",
                "🐮","🐷","🐸","🐵","🙈","🙉","🙊","🐔","🐧","🐦","🐤","🦄",
                "🐝","🦋","🐌","🐞","🐢","🐍","🦎","🦖","🐙","🦑","🦀","🐠",
                "🐟","🐬","🐳","🦈","🐊","🐅","🐆","🦓","🦍","🦧","🐘","🦒",
                "🌱","🌿","☘️","🍀","🎍","🪴","🌵","🌴","🌳","🌲","🌺","🌸",
                "🌼","🌻","🌞","🌝","🌚","🌈","☀️","⛅","🌧️","❄️","☃️","🌊"
            ]
        },
        {
            id: "food",
            icon: "🍔",
            name: groupName("食物与饮品", "Food & Drink"),
            emojis: [
                "🍏","🍎","🍐","🍊","🍋","🍌","🍉","🍇","🍓","🫐","🍈","🍒",
                "🍑","🥭","🍍","🥥","🥝","🍅","🥑","🥦","🌽","🥕","🧄","🍄",
                "🍞","🥐","🥖","🧀","🥚","🍳","🥞","🧇","🥓","🍗","🍖","🍔",
                "🍟","🍕","🌭","🥪","🌮","🌯","🥗","🍝","🍜","🍲","🍛","🍣",
                "🍱","🥟","🍤","🍙","🍚","🍘","🍥","🥠","🍡","🍧","🍨","🍦",
                "🥧","🧁","🍰","🎂","🍮","🍭","🍬","🍫","🍿","🍩","🍪","☕",
                "🧋","🥤","🍺","🍻","🥂","🍷","🍸","🍹","🧃","🧊","🥢","🍽️"
            ]
        },
        {
            id: "activities",
            icon: "⚽",
            name: groupName("活动与旅行", "Activities & Travel"),
            emojis: [
                "⚽","🏀","🏈","⚾","🥎","🎾","🏐","🏉","🥏","🎱","🏓","🏸",
                "🏒","🏑","🥍","🏏","⛳","🏹","🎣","🥊","🥋","🎽","🛹","🛼",
                "🛷","⛸️","🎿","🏂","🏋️","🤸","⛹️","🤺","🏊","🚴","🏆","🥇",
                "🎮","🕹️","🎲","♟️","🎯","🎳","🎸","🎹","🥁","🎤","🎧","🎬",
                "🚗","🚕","🚌","🚑","🚒","🚲","✈️","🚀","🛸","🚁","⛵","🚢",
                "🏠","🏢","🏥","🏦","🏨","🏪","🗼","🗽","🎡","🏖️","🏝️","🌋"
            ]
        },
        {
            id: "objects",
            icon: "💡",
            name: groupName("物品", "Objects"),
            emojis: [
                "⌚","📱","💻","⌨️","🖥️","🖨️","🖱️","📷","📸","🎥","📺","📻",
                "⏰","⌛","🔋","🔌","💡","🔦","🕯️","🧯","💸","💵","💳","💎",
                "⚖️","🧰","🔧","🔨","🛠️","🧲","🔫","🛡️","🔮","🧿","💊","🩹",
                "🧼","🪥","🧻","🧹","🧺","🧸","🎁","🎈","✉️","📩","📨","📧",
                "📌","📍","📎","📝","📁","📅","📊","📈","📉","🔒","🔑","🔔"
            ]
        }
    ];
}
function escapeChatHtml(value) {
    return String(value == null ? "" : value)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}

function safeChatResourceUrl(value, baseUrl) {
    var src = String(value || "").trim();
    // Uploaded chat media must remain below the controlled upload directory.
    if (!/^\/?static\/upload\/[a-zA-Z0-9_./-]+$/i.test(src) || src.indexOf("..") !== -1) {
        return "";
    }
    return escapeChatHtml(String(baseUrl || "") + (baseUrl && src.charAt(0) !== "/" ? "/" : "") + src);
}

function safeChatLinkUrl(value) {
    var src = String(value || "").trim();
    if (!/^(https?:\/\/|\/)[^\s]+$/i.test(src) || /^\/\//.test(src) || src.indexOf("\\") !== -1) {
        return "";
    }
    return escapeChatHtml(src);
}

function replaceContent(content, baseUrl) {
    return replaceSpecialTag(content, baseUrl);
}

// Render only the small, controlled chat markup supported by this application.
// All original message HTML is escaped before token expansion, preventing stored
// XSS through v-html while preserving images, audio, faces, links and newlines.
function replaceSpecialTag(value, baseUrl) {
    var faces = placeFace();
    var str = escapeChatHtml(value);
    baseUrl = typeof baseUrl === "undefined" ? "" : baseUrl;

    str = str.replace(/face\[([^\s\[\]]+?)\]/g, function (token, faceName) {
        var facePath = faces["[" + faceName + "]"];
        if (!facePath) {
            return token;
        }
        var alt = escapeChatHtml(faceName);
        var src = escapeChatHtml(String(baseUrl || "") + facePath);
        return '<img alt="' + alt + '" title="' + alt + '" src="' + src + '">';
    }).replace(/img\[([^\s\[\]]+?)\]/g, function (token, rawSrc) {
        var src = safeChatResourceUrl(rawSrc, baseUrl);
        if (!src) {
            return token;
        }
        return '<a href="' + src + '" target="_blank" rel="noopener noreferrer"><img data-src="' + src + '" data-lightbox class="chatImagePic" src="' + src + '?width=400"></a>';
    }).replace(/audio\[([^\s\[\]]+?)\]/g, function (token, rawSrc) {
        var src = safeChatResourceUrl(rawSrc, baseUrl);
        if (!src) {
            return token;
        }
        return '<div class="chatAudio"><audio controls src="' + src + '" class="audio"></audio></div>';
    }).replace(/file\[([^\s\[\]]+?)\]/g, function (token, rawSrc) {
        var src = safeChatResourceUrl(rawSrc, baseUrl);
        if (!src) {
            return token;
        }
        return '<a class="folderBtn" href="' + src + '" target="_blank" rel="noopener noreferrer" aria-label="下载历史附件"></a>';
    }).replace(/\[([^\]]+?)\]+link\[([^\]]+?)\]/g, function (token, text, rawSrc) {
        var src = safeChatLinkUrl(rawSrc);
        if (!src) {
            return text;
        }
        return '<a href="' + src + '" target="_blank" rel="noopener noreferrer">' + text + '</a>';
    }).replace(/(?:\r\n|\r|\n|\\n)/g, '<br>');

    return str;
}

// Parse rich text in an inert template, then rebuild only supported markup.
function renderRichChatMessage(content, baseUrl) {
    var raw = String(content == null ? "" : content);
    if (!/<\/?(?:p|br|div|strong|b|em|i|u|s|strike|ul|ol|li|blockquote|h[1-6]|a|img|span)(?=[\s/>])/i.test(raw)) {
        return replaceSpecialTag(raw, baseUrl);
    }
    var parsed = document.createElement("template");
    parsed.innerHTML = raw;
    var allowed = new Set(["p", "br", "div", "strong", "b", "em", "i", "u", "s", "strike",
        "ul", "ol", "li", "blockquote", "h1", "h2", "h3", "h4", "h5", "h6"]);
    function render(node, depth) {
        if (depth > 40) return "";
        if (node.nodeType === 3) return replaceSpecialTag(node.nodeValue, baseUrl);
        if (node.nodeType !== 1 || node.namespaceURI !== "http://www.w3.org/1999/xhtml") return "";
        var tag = node.localName;
        if (tag === "img") {
            var path = node.getAttribute("src") || "";
            var src = /\.(?:png|jpe?g|gif|webp)$/i.test(path) && safeChatResourceUrl(path, baseUrl);
            return src ? '<img class="chatImagePic" src="' + src + '" alt="' +
                escapeChatHtml(node.getAttribute("alt")) + '">' : "";
        }
        if (tag === "br") return "<br>";
        if (!allowed.has(tag) && tag !== "span" && tag !== "a") return "";
        var children = Array.from(node.childNodes, function (child) { return render(child, depth + 1); }).join("");
        if (tag === "span") return children;
        if (tag === "a") {
            var href = safeChatLinkUrl(node.getAttribute("href"));
            return href ? '<a href="' + href + '" target="_blank" rel="noopener noreferrer">' + children + '</a>' : children;
        }
        return "<" + tag + ">" + children + "</" + tag + ">";
    }
    return Array.from(parsed.content.childNodes, function (node) { return render(node, 0); }).join("");
}

function acceptChatParentEvent(event) {
    if (window.parent === window || event.source !== window.parent || !document.referrer ||
        !event.data || typeof event.data.content !== "string") return false;
    try {
        var parent = new URL(document.referrer);
        return /^(https?:)$/.test(parent.protocol) && event.origin === parent.origin;
    } catch (ignored) { return false; }
}

function renderChatMessage(content) { return renderRichChatMessage(content); }

// Reconcile persisted unread messages whenever the visitor websocket opens.
// WebSocket writes can fail during a short mobile-network transition even
// though the message was safely stored by the server.
function syncUnreadVisitorMessages(vm) {
    if (!vm || !vm.visitor || !vm.visitor.visitor_id) {
        return;
    }
    var entId = typeof ENT_ID === "undefined" ? "" : ENT_ID;
    $.get("/2/messages_page", {visitor_id: vm.visitor.visitor_id, ent_id: entId, page: 1, pagesize: 50}, function (res) {
        if (!res || res.code !== 200 || !res.result || !Array.isArray(res.result.list)) {
            return;
        }
        mergeRecoveredVisitorMessages(vm, res.result.list);
    });
    $.get("/2/messages_unread", {visitor_id: vm.visitor.visitor_id, ent_id: entId}, function (res) {
        if (!res || res.code !== 200 || !Array.isArray(res.result)) {
            return;
        }
        mergeRecoveredVisitorMessages(vm, res.result);
    });
}

function mergeRecoveredVisitorMessages(vm, messages) {
    if (!vm || !Array.isArray(messages)) {
        return;
    }
    var added = false;
    for (var j = 0; j < messages.length; j++) {
        var message = messages[j];
        if (message.mes_type === "visitor") {
            continue;
        }
        var content = {
            avator: message.avator,
            name: message.name,
            content: message.content,
            is_kefu: false,
            is_reply: true,
            msg_id: message.msg_id,
            time: message.time,
            read_status: message.read_status === "read" ? GOFLY_LANG[LANG].read : GOFLY_LANG[LANG].unread
        };
        if (insertVisitorChatMessageByID(vm, content)) {
            added = true;
        }
    }
    if (added || hasUnreadVisitorMessages(vm)) {
        vm.haveUnreadMessage = true;
    }
    if (added) {
        vm.scrollBottom();
    }
}

function hasVisitorChatMessageID(vm, messageID) {
    if (!vm || !Array.isArray(vm.msgList) || !messageID) {
        return false;
    }
    for (var i = 0; i < vm.msgList.length; i++) {
        if (String(vm.msgList[i].msg_id) === String(messageID)) {
            return true;
        }
    }
    return false;
}

function insertVisitorChatMessageByID(vm, message) {
    if (!vm || !Array.isArray(vm.msgList) || !message) {
        return false;
    }
    if (!message.msg_id) {
        vm.msgList.push(message);
        return true;
    }
    if (hasVisitorChatMessageID(vm, message.msg_id)) {
        return false;
    }
    for (var i = 0; i < vm.msgList.length; i++) {
        var existingID = Number(vm.msgList[i].msg_id || 0);
        if (existingID > 0 && existingID > Number(message.msg_id)) {
            vm.msgList.splice(i, 0, message);
            return true;
        }
    }
    vm.msgList.push(message);
    return true;
}

function hasUnreadVisitorMessages(vm) {
    if (!vm || !Array.isArray(vm.msgList)) {
        return false;
    }
    for (var i = 0; i < vm.msgList.length; i++) {
        var message = vm.msgList[i];
        if (message.is_kefu === false && message.msg_id && message.read_status !== GOFLY_LANG[LANG].read) {
            return true;
        }
    }
    return false;
}

function visibleUnreadVisitorMessageIDs(vm) {
    if (!vm || !Array.isArray(vm.msgList)) {
        return [];
    }
    var ids = [];
    var seen = {};
    for (var i = 0; i < vm.msgList.length; i++) {
        var message = vm.msgList[i];
        if (message.is_kefu !== false || !message.msg_id || message.read_status === GOFLY_LANG[LANG].read || seen[String(message.msg_id)]) {
            continue;
        }
        seen[String(message.msg_id)] = true;
        ids.push(message.msg_id);
        if (ids.length === 200) {
            break;
        }
    }
    return ids;
}

function markVisitorMessagesRead(vm, messageIDs) {
    if (!vm || !Array.isArray(vm.msgList) || !Array.isArray(messageIDs)) {
        return;
    }
    var acknowledged = {};
    for (var i = 0; i < messageIDs.length; i++) {
        acknowledged[String(messageIDs[i])] = true;
    }
    for (var j = 0; j < vm.msgList.length; j++) {
        if (acknowledged[String(vm.msgList[j].msg_id)]) {
            vm.msgList[j].read_status = GOFLY_LANG[LANG].read;
        }
    }
    vm.haveUnreadMessage = hasUnreadVisitorMessages(vm);
}
function bigPic(src,isVisitor){
    alert(src);
    if (isVisitor) {
        window.open(src);
        return;
    }
}
function filter (obj){
    var imgType = ["image/jpeg","image/png","image/jpg","image/gif","image/webp"];
    var filetypes = imgType;
    var isnext = false;
    for (var i = 0; i < filetypes.length; i++) {
        if (filetypes[i] == obj.type) {
            return true;
        }
    }
    return false;
}
function sleep(time) {
    var startTime = new Date().getTime() + parseInt(time, 10);
    while(new Date().getTime() < startTime) {}
}
function checkLang(){
    var langs=["cn","en","jp"];
    var lang=getQuery("lang");
    if(lang!=""&&langs.indexOf(lang) > 0 ){
        return lang;
    }
    var lang=getLocalStorage("lang");
    if(lang){
        return lang;
    }
    return "cn";
}
function changeURLPar(destiny, par, par_value) {
    var pattern = par + '=([^&]*)';
    var replaceText = par + '=' + par_value;
    if (destiny.match(pattern)) {
        var tmp = '/\\' + par + '=[^&]*/';
        tmp = destiny.replace(eval(tmp), replaceText);
        return (tmp);
    }
    else {
        if (destiny.match('[\?]')) {
            return destiny + '&' + replaceText;
        }
        else {
            return destiny + '?' + replaceText;
        }
    }
    return destiny + '\n' + par + '\n' + par_value;
}
function getQuery(key) {
    var query = window.location.search.substring(1);
    var key_values = query.split("&");
    var params = {};
    key_values.map(function (key_val){
        var key_val_arr = key_val.split("=");
        params[key_val_arr[0]] = key_val_arr[1];
    });
    if(typeof params[key]!="undefined"){
        return params[key];
    }
    return "";
}
function utf8ToB64(str) {
    return window.btoa(unescape(encodeURIComponent(str)));
}
function b64ToUtf8(str) {
    return decodeURIComponent(escape(window.atob(str)));
}
//存储localStorge
function setLocalStorage(key,obj){
    if(!navigator.cookieEnabled||typeof window.localStorage == 'undefined'){
        return false;
    }
    localStorage.setItem(key, JSON.stringify(obj));
    return true;
}
//读取localStorge
function getLocalStorage(key){
    if(!navigator.cookieEnabled||typeof window.localStorage == 'undefined'){
        return false;
    }
    var str=localStorage.getItem(key);
    if(!str){
        return false;
    }
    return JSON.parse(str);
}
var imgs = document.querySelectorAll('img');

//offsetTop是元素与offsetParent的距离，循环获取直到页面顶部
function getTop(e) {
    var T = e.offsetTop;
    while(e = e.offsetParent) {
        T += e.offsetTop;
    }
    return T;
}

function lazyLoad(imgs) {
    var H = document.documentElement.clientHeight;//获取可视区域高度
    var S = document.documentElement.scrollTop || document.body.scrollTop;
    for (var i = 0; i < imgs.length; i++) {
        if (H + S > getTop(imgs[i])) {
            console.log(imgs[i]);
            imgs[i].src = imgs[i].getAttribute('data-src');
        }
    }
}
function loadImage(url){
    var image = new Image();
    image.src = url;
    console.log(image);
}
function image2Canvas(image) {
    var canvas = document.createElement('canvas')
    var ctx = canvas.getContext('2d')
    canvas.width = image.naturalWidth
    canvas.height = image.naturalHeight
    ctx.drawImage(image, 0, 0, canvas.width, canvas.height)
    return canvas
}
function canvas2DataUrl(canvas, quality, type) {
    return canvas.toDataURL(type || 'image/jpeg', quality || 0.8)
}
function dataUrl2Image(dataUrl, callback) {
    var image = new Image()
    image.onload = function() {
        callback(image)
    }
    image.src = dataUrl
}
function dateFormat(fmt, date) {
    let ret;
    const opt = {
        "Y+": date.getFullYear().toString(),        // 年
        "m+": (date.getMonth() + 1).toString(),     // 月
        "d+": date.getDate().toString(),            // 日
        "H+": date.getHours().toString(),           // 时
        "M+": date.getMinutes().toString(),         // 分
        "S+": date.getSeconds().toString()          // 秒
        // 有其他格式化字符需求可以继续添加，必须转化成字符串
    };
    for (let k in opt) {
        if(k!="Y+"){
            var length=opt[k].length;
            if(length<2){
                opt[k]="0"+opt[k];
            }
        }
        ret = new RegExp("(" + k + ")").exec(fmt);
        if (ret) {
            fmt = fmt.replace(ret[1], (ret[1].length == 1) ? (opt[k]) : (opt[k].padStart(ret[1].length, "0")))
        };
    };
    return fmt;
}

/**
 * 按系统配置时区显示固定日期时间，不使用“几分钟前”等相对时间。
 * 无时区的数据库时间字符串已经是系统时区，直接规范化即可。
 */
function formatSystemDateTime(value) {
    if (value === null || value === undefined || value === "") {
        return "";
    }

    var text = String(value).trim();
    var plainDateTime = text.match(/^(\d{4})-(\d{1,2})-(\d{1,2})[ T](\d{1,2}):(\d{1,2}):(\d{1,2})(?:\.\d+)?$/);
    if (plainDateTime) {
        return plainDateTime[1] + "-" +
            plainDateTime[2].padStart(2, "0") + "-" +
            plainDateTime[3].padStart(2, "0") + " " +
            plainDateTime[4].padStart(2, "0") + ":" +
            plainDateTime[5].padStart(2, "0") + ":" +
            plainDateTime[6].padStart(2, "0");
    }

    var date = value instanceof Date ? value : new Date(value);
    if (Number.isNaN(date.getTime())) {
        return text;
    }

    var timezoneMeta = document.querySelector('meta[name="system-timezone"]');
    var timezone = timezoneMeta ? timezoneMeta.getAttribute("content") : "Asia/Shanghai";
    try {
        var parts = new Intl.DateTimeFormat("zh-CN", {
            timeZone: timezone || "Asia/Shanghai",
            year: "numeric",
            month: "2-digit",
            day: "2-digit",
            hour: "2-digit",
            minute: "2-digit",
            second: "2-digit",
            hourCycle: "h23"
        }).formatToParts(date);
        var values = {};
        parts.forEach(function (part) {
            values[part.type] = part.value;
        });
        return values.year + "-" + values.month + "-" + values.day + " " +
            values.hour + ":" + values.minute + ":" + values.second;
    } catch (error) {
        return dateFormat("Y-m-d H:M:S", date);
    }
}
/**
 * 人性化时间
 * @param {Object} timestamp
 */
function beautifyTime(timestamp,lang){
    var mistiming = Math.round(new Date() / 1000)-timestamp;
    mistiming = Math.abs(mistiming)
    if(lang=="en"){
        var postfix = mistiming>0 ? 'ago' : 'later'
        var arrr = [' years ',' months ',' weeks ',' days ',' hours ',' minutes ',' seconds '];
        var just='just now';
    }else{
        var postfix = mistiming>0 ? '前' : '后'
        var arrr = ['年','个月','周','天','小时','分钟','秒'];
        var just='刚刚';
    }
    if(mistiming<=1){
        return just;
    }

    var arrn = [31536000,2592000,604800,86400,3600,60,1];

    for(var i=0; i<7; i++){
        var inm = Math.floor(mistiming/arrn[i])
        if(inm!=0){
            return inm+arrr[i] + postfix
        }
    }
}

/**
 * 判断是否是手机访问
 * @returns {boolean}
 */
function isMobile() {
    if( /Mobile|Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent) ) {
        return true;
    }
    return false;
}
//发送ajax的助手函数
function sendAjax(url,method,params,callback){
    $.ajax({
        type: method,
        url: url,
        data:params,
        success: function(data) {
            callback(data);
        }
    });
}
//复制文本
function copyText(text) {
    var target = document.createElement('input') //创建input节点
    target.value = text // 给input的value赋值
    document.body.appendChild(target) // 向页面插入input节点
    target.select() // 选中input
    document.execCommand("copy"); // 执行浏览器复制命令
    document.body.removeChild(target);
    return true;
}
function MyHereDoc(){
    /*HERE

    HERE*/
    var here = "HERE";
    var reobj = new RegExp("/\\*"+here+"\\n[\\s\\S]*?\\n"+here+"\\*/", "m");
    str = reobj.exec(MyHereDoc).toString();
    str = str.replace(new RegExp("/\\*"+here+"\\n",'m'),'').toString();
    return str.replace(new RegExp("\\n"+here+"\\*/",'m'),'').toString();
}
//js获取当前时间
function getNowDate() {
    return formatSystemDateTime(new Date());
}
//获取当前时间戳
function getTimestamp() {
    return new Date(getNowDate()).getTime();
}
//删除对象中的空属性
function removePropertyOfNull(obj){
    var i = obj.length;
    while (i--) {
        if (obj[i]===null) {
            obj.splice(i, 1);
        }
    }
    return obj;
}
//判断版本号大小
function compareVersion(v1, v2) {
    if (v1 == v2) {
        return 0;
    }

    const vs1 = v1.split(".").map(a => parseInt(a));
    const vs2 = v2.split(".").map(a => parseInt(a));

    const length = Math.min(vs1.length, vs2.length);
    for (let i = 0; i < length; i++) {
        if (vs1[i] > vs2[i]) {
            return 1;
        } else if (vs1[i] < vs2[i]) {
            return -1;
        }
    }

    if (length == vs1.length) {
        return -1;
    } else {
        return 1;
    }
}
function isWeiXin(){
    //window.navigator.userAgent属性包含了浏览器类型、版本、操作系统类型、浏览器引擎类型等信息，这个属性可以用来判断浏览器类型
    var ua = window.navigator.userAgent.toLowerCase();
    //通过正则表达式匹配ua中是否含有MicroMessenger字符串
    if(ua.match(/MicroMessenger/i) == 'micromessenger'){
        return true;
    }else{
        return false;
    }
}
;
