import { useEffect, useRef, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { PlusIcon, SendIcon, MessageSquareIcon, TrashIcon, LinkIcon, BrainCircuitIcon, PanelLeftIcon } from "lucide-react";
import toast from "react-hot-toast";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Input } from "@/components/ui/input";
import MemoContent from "@/components/MemoContent";
import { aiService, AIChatSession, AIChatMessage } from "@/utils/aiService";
import { cn } from "@/lib/utils";
import { MemoViewContext } from "@/components/MemoView/MemoViewContext";
import useCurrentUser from "@/hooks/useCurrentUser";
import type { Memo } from "@/types/proto/api/v1/memo_service_pb";

const Chat = () => {
    const { uid } = useParams();
    const navigate = useNavigate();

    const [sessions, setSessions] = useState<AIChatSession[]>([]);
    const [messages, setMessages] = useState<AIChatMessage[]>([]);
    const [input, setInput] = useState("");
    const [tagFilter, setTagFilter] = useState("");
    const [isSidebarOpen, setSidebarOpen] = useState(false);
    const [isGenerating, setIsGenerating] = useState(false);

    // Streaming state
    const [streamedResponse, setStreamedResponse] = useState("");
    const [activeTool, setActiveTool] = useState<{ name: string, input: string } | null>(null);
    const [sources, setSources] = useState<{ memo_uid: string, snippet: string }[]>([]);

    const currentUser = useCurrentUser();

    // Mock MemoViewContext for AI messages to safely render Tags and Mentions
    const mockMemoContextValue = {
        memo: { name: "memos/chat-mock", uid: "chat-mock", content: "", displayTime: new Date() } as unknown as Memo,
        creator: currentUser,
        currentUser: currentUser,
        parentPage: "/chat",
        isArchived: false,
        readonly: true,
        showNSFWContent: true,
        nsfw: false,
    };

    const messagesEndRef = useRef<HTMLDivElement>(null);

    const loadSessions = async () => {
        try {
            const data = await aiService.listSessions();
            setSessions(data);
        } catch (e: any) {
            toast.error(e.message);
        }
    };

    useEffect(() => {
        loadSessions();
    }, []);

    useEffect(() => {
        if (uid) {
            // Load messages for session
            aiService.loadMessages(uid).then(setMessages).catch((e: any) => toast.error(e.message));
            setStreamedResponse("");
            setActiveTool(null);
            setSources([]);
        } else {
            setMessages([]);
        }
    }, [uid]);

    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }, [messages, streamedResponse]);

    const handleCreateSession = async () => {
        try {
            const sess = await aiService.createSession();
            setSessions([sess, ...sessions]);
            navigate(`/chat/${sess.uid}`);
            if (window.innerWidth < 768) setSidebarOpen(false);
        } catch (e: any) {
            toast.error(e.message);
        }
    };

    const handleDeleteSession = async (e: React.MouseEvent, delUid: string) => {
        e.stopPropagation();
        if (!confirm("Delete this chat?")) return;
        try {
            await aiService.deleteSession(delUid);
            setSessions(sessions.filter(s => s.uid !== delUid));
            if (uid === delUid) {
                navigate("/chat");
            }
        } catch (e: any) {
            toast.error(e.message);
        }
    };

    const handleSend = async () => {
        if (!input.trim() || isGenerating) return;
        const txt = input.trim();
        setInput("");
        setIsGenerating(true);

        let currentSessionUid = uid;
        if (!currentSessionUid) {
            try {
                const sess = await aiService.createSession();
                setSessions([sess, ...sessions]);
                currentSessionUid = sess.uid;
                // Don't navigate yet, otherwise effect will clear things
                window.history.pushState({}, "", `/chat/${currentSessionUid}`);
            } catch (e: any) {
                toast.error(e.message);
                setIsGenerating(false);
                return;
            }
        }

        // Optimistically add user message
        setMessages(prev => [...prev, {
            id: Date.now(),
            role: "user",
            content: txt,
            createdTs: Date.now() / 1000,
        }]);

        try {
            setStreamedResponse("");
            setSources([]);
            setActiveTool(null);

            const gen = aiService.chat(currentSessionUid, txt, tagFilter);
            let contentAcc = "";

            for await (const event of gen) {
                if (event.type === "token" && event.content) {
                    contentAcc += event.content;
                    setStreamedResponse(contentAcc);
                } else if (event.type === "tool_call" && event.payload) {
                    setActiveTool({ name: event.payload.name, input: event.payload.input });
                } else if (event.type === "source" && event.payload) {
                    setSources(prev => [...prev, event.payload as any]);
                } else if (event.type === "error" && event.content) {
                    toast.error("AI Error: " + event.content);
                } else if (event.type === "done") {
                    // done
                }
            }

            // Finalise message
            setMessages(prev => [...prev, {
                id: Date.now(),
                role: "assistant",
                content: contentAcc,
                createdTs: Date.now() / 1000,
            }]);
            setStreamedResponse("");
            setActiveTool(null);
            loadSessions(); // Reload titles (might have auto-titled)
        } catch (e: any) {
            toast.error(e.message);
        } finally {
            setIsGenerating(false);
        }
    };

    return (
        <section className="w-full h-[100dvh] flex flex-row justify-start items-start bg-background text-foreground overflow-hidden">
            {/* Sidebar */}
            <div className={cn(
                "fixed md:relative z-20 md:z-auto h-full w-64 flex flex-col bg-background border-r border-border transition-transform duration-300 shadow-xl md:shadow-none",
                isSidebarOpen ? "translate-x-0" : "-translate-x-full md:translate-x-0"
            )}>
                <div className="p-4 flex items-center justify-between border-b">
                    <Button onClick={handleCreateSession} variant="outline" className="w-full flex gap-2">
                        <PlusIcon className="w-4 h-4" /> New Chat
                    </Button>
                    <Button variant="ghost" size="icon" className="md:hidden ml-2 shrink-0 border" onClick={() => setSidebarOpen(false)}>
                        x
                    </Button>
                </div>
                <div className="flex-1 overflow-y-auto p-2 space-y-1">
                    {sessions.map(s => (
                        <div
                            key={s.uid}
                            onClick={() => { navigate(`/chat/${s.uid}`); if (window.innerWidth < 768) setSidebarOpen(false); }}
                            className={cn(
                                "group flex items-center justify-between p-2 rounded-lg cursor-pointer hover:bg-muted transition-colors",
                                uid === s.uid && "bg-muted font-medium"
                            )}
                        >
                            <div className="flex items-center gap-2 truncate text-sm">
                                <MessageSquareIcon className="w-4 h-4 shrink-0 opacity-70" />
                                <span className="truncate">{s.title || "New Chat"}</span>
                            </div>
                            <button onClick={(e) => handleDeleteSession(e, s.uid)} className="opacity-0 group-hover:opacity-100 p-1 hover:text-red-500 rounded transition-opacity">
                                <TrashIcon className="w-3.5 h-3.5" />
                            </button>
                        </div>
                    ))}
                </div>
            </div>

            {/* Main Chat Area */}
            <div className="flex-1 flex flex-col h-full overflow-hidden bg-background">
                <div className="flex items-center p-3 border-b border-border bg-background/50 backdrop-blur z-10 w-full shrink-0">
                    <Button variant="ghost" size="icon" className="md:hidden mr-2" onClick={() => setSidebarOpen(true)}>
                        <PanelLeftIcon className="w-5 h-5" />
                    </Button>
                    <h1 className="text-lg font-medium opacity-80">AI Chat</h1>
                    <Button variant="ghost" size="icon" className="ml-auto" onClick={() => navigate("/")}>
                        x
                    </Button>
                </div>

                <div className="flex-1 overflow-y-auto p-4 space-y-6">
                    {messages.length === 0 && !streamedResponse && (
                        <div className="h-full flex flex-col items-center justify-center text-center opacity-50">
                            <BrainCircuitIcon className="w-16 h-16 mb-4" />
                            <p className="text-lg">How can I help you today?</p>
                            <p className="text-sm">I can search your notes, answer questions, or browse the web.</p>
                        </div>
                    )}

                    {messages.map((m) => (
                        <div key={m.id} className={cn("flex w-full", m.role === "user" ? "justify-end" : "justify-start")}>
                            <div className={cn("max-w-[85%] rounded-2xl p-4 shadow-sm",
                                m.role === "user" ? "bg-primary text-primary-foreground" : "bg-card border border-border text-card-foreground"
                            )}>
                                {m.role === "assistant" ? (
                                    <MemoViewContext.Provider value={mockMemoContextValue}>
                                        <MemoContent content={m.content} />
                                    </MemoViewContext.Provider>
                                ) : (
                                    <div className="whitespace-pre-wrap">{m.content}</div>
                                )}
                            </div>
                        </div>
                    ))}

                    {(streamedResponse || activeTool) && (
                        <div className="flex w-full justify-start">
                            <div className="max-w-[85%] rounded-2xl p-4 shadow-sm bg-card border border-border text-card-foreground flex flex-col gap-2">
                                {activeTool && (
                                    <div className="text-xs flex items-center gap-2 text-muted-foreground bg-muted p-2 rounded w-fit">
                                        <BrainCircuitIcon className="w-3 h-3 animate-pulse" />
                                        Using {activeTool.name}...
                                    </div>
                                )}
                                {sources.length > 0 && (
                                    <div className="flex flex-wrap gap-2 mb-2">
                                        {sources.map((s, i) => (
                                            <a key={i} href={`/memos/${s.memo_uid}`} target="_blank" className="text-xs flex items-center gap-1 bg-muted text-foreground px-2 py-1 rounded-full hover:underline border border-border">
                                                <LinkIcon className="w-3 h-3" /> Memo {s.memo_uid}
                                            </a>
                                        ))}
                                    </div>
                                )}
                                <MemoViewContext.Provider value={mockMemoContextValue}>
                                    <MemoContent content={streamedResponse + " ▌"} />
                                </MemoViewContext.Provider>
                            </div>
                        </div>
                    )}

                    <div ref={messagesEndRef} />
                </div>

                <div className="p-4 bg-background border-t border-border flex flex-col gap-2 shrink-0">
                    <div className="flex gap-2 w-full max-w-4xl mx-auto items-end">
                        <Input
                            value={tagFilter}
                            onChange={e => setTagFilter(e.target.value)}
                            placeholder="#tag filter (opt)"
                            className="w-32 h-10"
                        />
                        <Textarea
                            value={input}
                            onChange={e => setInput(e.target.value)}
                            onKeyDown={e => {
                                if (e.key === 'Enter' && !e.shiftKey) {
                                    e.preventDefault();
                                    handleSend();
                                }
                            }}
                            placeholder="Message AI Assistant..."
                            className="flex-1 min-h-[40px] max-h-32 resize-none"
                            disabled={isGenerating}
                        />
                        <Button onClick={handleSend} disabled={!input.trim() || isGenerating} size="icon" className="h-[40px] w-[50px] px-2 shrink-0">
                            <SendIcon className="w-4 h-4" />
                        </Button>
                    </div>
                    <div className="text-center text-xs text-muted-foreground pb-2 max-w-4xl mx-auto">
                        AI can make mistakes. Check important info.
                    </div>
                </div>
            </div>
        </section>
    );
};

export default Chat;
