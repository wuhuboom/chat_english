const assert = require("node:assert/strict");
const fs = require("node:fs");

const source = fs.readFileSync("static/js/chat-main.js", "utf8");
const start = source.indexOf("        //粘贴图片进入待发送区");
const end = source.indexOf("        openUrl(url){", start);
assert.notEqual(start, -1, "paste draft handler must exist");
assert.notEqual(end, -1, "paste draft handler boundary must exist");

const pasteHandler = source.slice(start, end);
assert.equal(pasteHandler.includes("pendingImageFile=file"), true, "pasted image should become a draft");
assert.equal(pasteHandler.includes("pendingImagePreview=URL.createObjectURL(file)"), true, "draft should have a local preview");
assert.equal(pasteHandler.includes("chatToUser("), false, "pasting must not send the image");
assert.equal(pasteHandler.includes("$.ajax("), false, "pasting must not upload before confirmation");

const sendStart = source.indexOf("        chatToUser() {");
const sendEnd = source.indexOf("        sendChatMessage(", sendStart);
const sendHandler = source.slice(sendStart, sendEnd);
assert.equal(sendHandler.includes("if(this.pendingImageFile)"), true, "Enter or send button should handle the image draft");
assert.equal(sendHandler.includes("uploadPendingImageAndSend(messageContent)"), true, "confirmed draft should upload and send");

for (const template of [
    "static/templates/default/chat_main.html",
    "static/templates/default/chat_main1.html"
]) {
    const html = fs.readFileSync(template, "utf8");
    assert.match(html, /class="pastedImageDraft"/, template + " should show the image draft");
    assert.match(html, /@click="clearPendingImage"/, template + " should allow cancelling the draft");
}

console.log("chat paste draft tests passed");
