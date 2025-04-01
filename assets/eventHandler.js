export class EventHandler {
    constructor(canvasManager, webSocketClient, config) {
        this.canvasManager = canvasManager;
        this.webSocketClient = webSocketClient;
        this.config = config;
        this.inputBuffer = "";
        this.setupEvents();
    }

    setupEvents() {
        const canvas = this.canvasManager.canvas;

        canvas.addEventListener("click", (e) => {
            const rect = canvas.getBoundingClientRect();
            const cellWidth = rect.width / this.config.getBoardWidth();
            const cellHeight = rect.height / this.config.getBoardHeight();
            const x = Math.min(this.config.getBoardWidth() - 2, Math.max(1, Math.floor((e.clientX - rect.left) / cellWidth)));
            const y = Math.min(this.config.getBoardHeight() - 2, Math.max(1, Math.floor((e.clientY - rect.top) / cellHeight)));
            this.sendMessage({
                type: "birth",
                x: x,
                y: y,
                gameID: this.config.getGameID()
            });
        });

        document.addEventListener("keydown", (e) => {
            if (e.key === "Shift" || e.key === "Control" || e.key === "Alt") return;

            this.inputBuffer += e.key.toLowerCase();
            console.log("[EventHandler] Input buffer updated:", this.inputBuffer);

            const commands = [
                { input: "color:red", type: "setBackgroundColor", color: "red" },
                { input: "color:blue", type: "setBackgroundColor", color: "blue" },
                { input: "color:green", type: "setBackgroundColor", color: "green" },
                { input: "color:reset", type: "setBackgroundColor", color: "#111" },
                { input: "clear", type: "clear" },
                { input: "random", type: "randomBirth", percentage: 50 },
                { input: "stop", type: "stop" },
                { input: "resume", type: "resume" },
                { input: "slide", type: "pattern", pattern: "glider" },
                { input: "blink", type: "pattern", pattern: "blinker" },
                { input: "toad", type: "pattern", pattern: "toad" },
                { input: "pulse", type: "pattern", pattern: "pulsar" },
                { input: "gun", type: "pattern", pattern: "gosper_glider_gun" },
                { input: "pent", type: "pattern", pattern: "r_pentomino" },
                { input: "snark", type: "pattern", pattern: "snark" },
                { input: "engine", type: "pattern", pattern: "2_engine" },
                { input: "hilbert", type: "pattern", pattern: "david_hilbert" }
            ];

            for (const cmd of commands) {
                if (this.inputBuffer.includes(cmd.input)) {
                    const msg = { type: cmd.type, gameID: this.config.getGameID(), ...cmd };
                    delete msg.input;
                    this.sendMessage(msg);
                    this.inputBuffer = "";
                    break;
                }
            }

            if (this.inputBuffer.length > 20) {
                this.inputBuffer = this.inputBuffer.slice(-10);
                console.log("[EventHandler] Input buffer truncated:", this.inputBuffer);
            }
        });
    }

    sendMessage(message) {
        this.webSocketClient.send(message);
    }
}