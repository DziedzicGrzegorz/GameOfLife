import {CanvasManager} from './canvasManager.js';
import {WebSocketClient} from './webSocketClient.js';
import {GameState} from './gameState.js';
import {GameRenderer} from './gameRenderer.js';
import {EventHandler} from './eventHandler.js';
import {GameConfig} from './gameConfig.js';

class GameClient {
    constructor() {
        this.config = new GameConfig();
        this.canvasManager = new CanvasManager();
        this.webSocketClient = new WebSocketClient("ws://localhost:8080/ws");
        this.gameState = new GameState();
        this.gameRenderer = new GameRenderer(this.canvasManager);
        this.eventHandler = new EventHandler(this.canvasManager, this.webSocketClient, this.config);
    }

    init() {

        // Send init message immediately after WebSocket opens
        this.webSocketClient.onMessage((data) => {
            this.gameState.update(data);
            this.gameRenderer.render(this.gameState.getState());
        });

        // Ensure WebSocket is open before sending
        if (this.webSocketClient.ws.readyState === WebSocket.OPEN) {
            this.sendInitMessage();
        } else {
            this.webSocketClient.ws.onopen = () => {
                this.sendInitMessage();
            };
        }
    }

    sendInitMessage() {
        this.webSocketClient.send({
            type: "init",
            gameID: this.config.getGameID(),
            width: this.config.getBoardWidth(),
            height: this.config.getBoardHeight(),
            cellSize: this.config.getCellSize()
        });
    }
}

// Start the game client asynchronously
(async () => {
    const gameClient = new GameClient();
    await gameClient.init();
})();