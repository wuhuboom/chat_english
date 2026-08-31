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
        return Boolean(isEnter && (event.ctrlKey || event.metaKey));
    }

    return {
        normalizeMessage: normalizeMessage,
        shouldSendOnKeydown: shouldSendOnKeydown
    };
}));
