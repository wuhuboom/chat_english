const assert = require("node:assert/strict");
const ChatInput = require("../static/js/chat-input.js");

assert.equal(ChatInput.normalizeMessage("第一行\n第二行"), "第一行\n第二行");
assert.equal(ChatInput.normalizeMessage(" 第一行\r\n第二行 \n"), "第一行\n第二行");
assert.equal(ChatInput.normalizeMessage("\n\r\n"), "");

assert.equal(ChatInput.shouldSendOnKeydown({key: "Enter"}), false);
assert.equal(ChatInput.shouldSendOnKeydown({key: "Enter", ctrlKey: true}), true);
assert.equal(ChatInput.shouldSendOnKeydown({key: "Enter", metaKey: true}), true);
assert.equal(ChatInput.shouldSendOnKeydown({key: "Enter", ctrlKey: true, isComposing: true}), false);
assert.equal(ChatInput.shouldSendOnKeydown({key: "Enter", ctrlKey: true, keyCode: 229}), false);
assert.equal(ChatInput.shouldSendOnKeydown({key: "a", ctrlKey: true}), false);

console.log("chat input keyboard tests passed");
