export interface AIChatSession {
    uid: string;
    title: string;
    createdTs: number;
    updatedTs: number;
}

export interface AIChatMessage {
    id: number;
    role: "user" | "assistant" | "tool";
    content: string;
    toolName?: string;
    createdTs: number;
}

export interface AIChatEvent {
    type: "token" | "tool_call" | "source" | "done" | "error";
    content?: string;
    payload?: any;
}

export const aiService = {
    async listSessions(): Promise<AIChatSession[]> {
        const res = await fetch("/api/v1/ai/sessions", {
            headers: { "Content-Type": "application/json" },
        });
        if (!res.ok) throw new Error("Failed to list sessions");
        return res.json();
    },

    async createSession(title: string = "New Chat"): Promise<AIChatSession> {
        const res = await fetch("/api/v1/ai/sessions", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ title }),
        });
        if (!res.ok) throw new Error("Failed to create session");
        return res.json();
    },

    async renameSession(uid: string, title: string): Promise<AIChatSession> {
        const res = await fetch(`/api/v1/ai/sessions/${uid}`, {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ title }),
        });
        if (!res.ok) throw new Error("Failed to rename session");
        return res.json();
    },

    async deleteSession(uid: string): Promise<void> {
        const res = await fetch(`/api/v1/ai/sessions/${uid}`, {
            method: "DELETE",
        });
        if (!res.ok) throw new Error("Failed to delete session");
    },

    async loadMessages(uid: string): Promise<AIChatMessage[]> {
        const res = await fetch(`/api/v1/ai/sessions/${uid}/messages`);
        if (!res.ok) throw new Error("Failed to load messages");
        return res.json();
    },

    async *chat(uid: string, content: string, tagFilter: string = ""): AsyncGenerator<AIChatEvent, void, unknown> {
        const res = await fetch(`/api/v1/ai/sessions/${uid}/chat`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ content, tagFilter }),
        });

        if (!res.ok) {
            throw new Error("Failed to send chat message");
        }

        if (!res.body) {
            throw new Error("No response body");
        }

        const reader = res.body.getReader();
        const decoder = new TextDecoder("utf-8");
        let buffer = "";

        try {
            while (true) {
                const { value, done } = await reader.read();
                if (done) break;

                buffer += decoder.decode(value, { stream: true });

                while (buffer.includes("\n\n")) {
                    const eventEndIndex = buffer.indexOf("\n\n");
                    const eventLine = buffer.slice(0, eventEndIndex);
                    buffer = buffer.slice(eventEndIndex + 2);

                    if (eventLine.startsWith("data: ")) {
                        const dataStr = eventLine.slice(6);
                        if (dataStr === "[DONE]") {
                            return;
                        }
                        try {
                            const event: AIChatEvent = JSON.parse(dataStr);
                            yield event;
                        } catch (e) {
                            console.warn("Failed to parse SSE event", dataStr, e);
                        }
                    }
                }
            }
        } finally {
            reader.releaseLock();
        }
    }
};
