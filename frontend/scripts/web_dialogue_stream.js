// Loaded from the exported PCK by web_dialogue_stream.gd.
// One client per dialogue box; cancellation also invalidates late fetch results.
globalThis.TownDialogueStream = {
    create(callback) {
        let current = null;
        function cancel() {
            const request = current;
            current = null;
            if (request) {
                clearTimeout(request.timer);
                request.controller.abort();
            }
        }
        function start(url, bodyJSON, timeoutMs) {
            cancel();
            const body = JSON.parse(bodyJSON);
            const request = { controller: new AbortController(), timer: null };
            current = request;
            let finished = false;
            function emit(event, data) {
                if (current !== request || finished) return;
                if (event === "done" || event === "error") finished = true;
                callback(JSON.stringify({ request_id: body.request_id, event, data }));
            }
            request.timer = setTimeout(() => {
                emit("error", { message: "等待回复超时，请稍后再试。" });
                request.controller.abort();
            }, timeoutMs);
            // Keep callbacks outside the synchronous Godot -> JS start() call.
            void (async () => {
                let reader;
                try {
                    const response = await fetch(url, {
                        method: "POST",
                        headers: { "Content-Type": "application/json", Accept: "text/event-stream" },
                        credentials: "same-origin",
                        body: bodyJSON,
                        signal: request.controller.signal,
                    });
                    if (current !== request || finished) return;
                    if (!response.ok) throw new Error(`请求失败（HTTP ${response.status}），请稍后再试。`);
                    if (!(response.headers.get("content-type") || "").toLowerCase().includes("text/event-stream"))
                        throw new Error("接口没有返回 SSE 数据，请检查流式接口地址。");
                    if (!response.body) throw new Error("浏览器未提供流式响应。");
                    reader = response.body.getReader();
                    const decoder = new TextDecoder("utf-8", { fatal: true });
                    let line = "", eventName = "", dataLines = [], frameSize = 0, skipLF = false;
                    let replySize = 0;
                    function processLine() {
                        if (line === "") {
                            if (dataLines.length && ["delta", "done", "error"].includes(eventName)) {
                                const data = JSON.parse(dataLines.join("\n"));
                                if (!data || typeof data !== "object" || Array.isArray(data))
                                    throw new Error("回复事件格式错误。");
                                if (data.request_id != null && data.request_id !== body.request_id)
                                    throw new Error("回复的请求编号不匹配。");
                                if (data.npc_id != null && data.npc_id !== body.npc_id)
                                    throw new Error("回复的 NPC 编号不匹配。");
                                if (eventName === "delta") {
                                    if (typeof data.text !== "string") throw new Error("回复片段格式错误。");
                                    replySize += data.text.length;
                                    if (replySize > 1024 * 1024) throw new Error("回复过长，本轮已停止。");
                                }
                                if (eventName === "done" && typeof data.reply !== "string")
                                    throw new Error("完整回复格式错误。");
                                emit(eventName, data);
                            }
                            eventName = "";
                            dataLines = [];
                            frameSize = 0;
                        } else if (!line.startsWith(":")) {
                            const colon = line.indexOf(":");
                            const field = colon < 0 ? line : line.slice(0, colon);
                            let value = colon < 0 ? "" : line.slice(colon + 1);
                            if (value.startsWith(" ")) value = value.slice(1);
                            if (field === "event") eventName = value;
                            if (field === "data") dataLines.push(value);
                        }
                        line = "";
                    }
                    function feed(text) {
                        for (const char of text) {
                            if (finished || current !== request) break;
                            if (skipLF) {
                                skipLF = false;
                                if (char === "\n") continue;
                            }
                            if (++frameSize > 1024 * 1024) throw new Error("回复事件过长。");
                            if (char === "\r" || char === "\n") {
                                processLine();
                                skipLF = char === "\r";
                            } else {
                                line += char;
                            }
                        }
                    }
                    while (!finished && current === request) {
                        const chunk = await reader.read();
                        if (current !== request || finished) break;
                        feed(decoder.decode(chunk.value, { stream: !chunk.done }));
                        if (chunk.done) {
                            // SSE requires a blank line to complete an event.
                            if (!finished) throw new Error("连接已中断，未收到完整回复。");
                            break;
                        }
                    }
                } catch (error) {
                    emit("error", { message: typeof error?.message === "string" ? error.message : "网络请求失败。" });
                } finally {
                    clearTimeout(request.timer);
                    if (reader) void reader.cancel().catch(() => {});
                    request.controller.abort();
                    if (current === request) current = null;
                }
            })();
        }
        return { start, cancel };
    },
};
