export class WebSocketClient {
    constructor(url) {
        this.ws = new WebSocket(url);
        this.callbacks = [];

        this.ws.onopen = () => {
            console.log("[WebSocketClient] Connected to server");
        };

        this.ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.callbacks.forEach(callback => callback(data));
        };

        this.ws.onerror = (err) => {
            console.error("[WebSocketClient] Error:", err);
        };

        this.ws.onclose = (event) => {
            console.log("[WebSocketClient] Connection closed - Code:", event.code, "Reason:", event.reason);
        };
    }

    send(message) {
        if (this.ws.readyState === WebSocket.OPEN) {
            const msgString = JSON.stringify(message);
            console.log("[WebSocketClient] Sending message:", message);
            this.ws.send(msgString);
        } else {
            console.warn("[WebSocketClient] Cannot send message, WebSocket not open:", message);
        }
    }

    onMessage(callback) {
        this.callbacks.push(callback);
    }
}