const assert = require("node:assert/strict");
const ChatInput = require("../static/js/chat-input.js");

assert.equal(ChatInput.normalizeMessage("第一行\n第二行"), "第一行\n第二行");
assert.equal(ChatInput.normalizeMessage(" 第一行\r\n第二行 \n"), "第一行\n第二行");
assert.equal(ChatInput.normalizeMessage("\n\r\n"), "");

assert.equal(ChatInput.shouldSendOnKeydown({key: "Enter"}), true);
assert.equal(ChatInput.shouldSendOnKeydown({key: "Enter", ctrlKey: true}), false);
assert.equal(ChatInput.shouldSendOnKeydown({key: "Enter", metaKey: true}), false);
assert.equal(ChatInput.shouldSendOnKeydown({key: "Enter", isComposing: true}), false);
assert.equal(ChatInput.shouldSendOnKeydown({key: "Enter", ctrlKey: true, keyCode: 229}), false);
assert.equal(ChatInput.shouldSendOnKeydown({key: "a", ctrlKey: true}), false);

assert.equal(ChatInput.shouldInsertNewlineOnKeydown({key: "Enter", ctrlKey: true}), true);
assert.equal(ChatInput.shouldInsertNewlineOnKeydown({key: "Enter", metaKey: true}), true);
assert.equal(ChatInput.shouldInsertNewlineOnKeydown({key: "Enter"}), false);
assert.equal(ChatInput.shouldInsertNewlineOnKeydown({key: "Enter", ctrlKey: true, isComposing: true}), false);

assert.deepEqual(
    ChatInput.insertNewlineAtSelection("第一行第二行", 3, 3),
    {content: "第一行\n第二行", cursorPosition: 4}
);
assert.deepEqual(
    ChatInput.insertNewlineAtSelection("第一行旧内容第二行", 3, 6),
    {content: "第一行\n第二行", cursorPosition: 4}
);

console.log("chat input keyboard tests passed");
