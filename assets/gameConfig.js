export class GameConfig {
    constructor() {
        this.cellSize = 5; // Adjustable cell density
        this.boardWidth = Math.floor(window.innerWidth / this.cellSize);
        this.boardHeight = Math.floor(window.innerHeight / this.cellSize);
        this.step = 50;
        this.minInterval = 50;
        this.maxInterval = 2000;

        const pathParts = window.location.pathname.split('/');
        this.gameID = pathParts.length > 1 && pathParts[1] ? pathParts[1] : `game_${Date.now().toString(36)}_${Math.random().toString(36).substr(2, 5)}`;
        console.log("[GameConfig] Initialized with gameID:", this.gameID,
            "Board dimensions:", this.boardWidth, "x", this.boardHeight,
            "Source:", pathParts.length > 1 && pathParts[1] ? "URL" : "Generated");
    }

    getCellSize() {
        return this.cellSize;
    }

    getBoardWidth() {
        return this.boardWidth;
    }

    getBoardHeight() {
        return this.boardHeight;
    }

    getGameID() {
        return this.gameID;
    }


    getStep() {
        return this.step;
    }

    getMinInterval() {
        return this.minInterval;
    }

    getMaxInterval() {
        return this.maxInterval;
    }
}