(function (root, factory) {
    var api = factory();
    if (typeof module === "object" && module.exports) {
        module.exports = api;
        return;
    }
    root.ChatInput = api;
}(typeof self !== "undefined" ? self : this, function () {
    function normalizeMessage(content) {
        return String(content || "")
            .replace(/\r\n?/g, "\n")
            .trim();
    }

    function shouldSendOnKeydown(event) {
        if (!event || event.isComposing || event.keyCode === 229) {
            return false;
        }
        var isEnter = event.key === "Enter" || event.keyCode === 13;
        return Boolean(isEnter && !event.ctrlKey && !event.metaKey);
    }

    function shouldInsertNewlineOnKeydown(event) {
        if (!event || event.isComposing || event.keyCode === 229) {
            return false;
        }
        var isEnter = event.key === "Enter" || event.keyCode === 13;
        return Boolean(isEnter && (event.ctrlKey || event.metaKey));
    }

    function insertNewlineAtSelection(content, selectionStart, selectionEnd) {
        var value = String(content || "");
        var start = Number.isInteger(selectionStart) ? selectionStart : value.length;
        var end = Number.isInteger(selectionEnd) ? selectionEnd : start;
        start = Math.max(0, Math.min(start, value.length));
        end = Math.max(start, Math.min(end, value.length));

        return {
            content: value.slice(0, start) + "\n" + value.slice(end),
            cursorPosition: start + 1
        };
    }

    return {
        normalizeMessage: normalizeMessage,
        shouldSendOnKeydown: shouldSendOnKeydown,
        shouldInsertNewlineOnKeydown: shouldInsertNewlineOnKeydown,
        insertNewlineAtSelection: insertNewlineAtSelection
    };
}));
