export class GameRenderer {
    constructor(canvasManager) {
        this.canvasManager = canvasManager;
    }

    render(gameState) {
        const ctx = this.canvasManager.getContext();
        const width = this.canvasManager.getWidth();
        const height = this.canvasManager.getHeight();

        console.log("[GameRenderer] Rendering - BackgroundColor:", gameState.BackgroundColor, "Color:", gameState.Color);
        ctx.fillStyle = gameState.BackgroundColor;
        ctx.fillRect(0, 0, width, height);
        ctx.fillStyle = gameState.Color;

        const cellWidth = width / gameState.Width;
        const cellHeight = height / gameState.Height;
        console.log("[GameRenderer] Board dimensions:", gameState.Width, "x", gameState.Height);
        console.log("[GameRenderer] Cell dimensions:", cellWidth, "x", cellHeight);

        let liveCells = 0;
        for (let y = 1; y < gameState.Height - 1; y++) {
            for (let x = 1; x < gameState.Width - 1; x++) {
                if (gameState.Board[y][x] >= 100) {
                    liveCells++;
                    ctx.fillRect((x - 1) * cellWidth, (y - 1) * cellHeight, cellWidth, cellHeight);
                }
            }
        }
        console.log("[GameRenderer] Number of live cells rendered:", liveCells);

        if (gameState.Stopped && liveCells === 0) {
            console.log("[GameRenderer] Game stopped: No live cells remaining");
            ctx.fillStyle = "white";
            ctx.font = `${Math.min(width, height) / 20}px Arial`;
            ctx.textAlign = "center";
            ctx.fillText("Game Over: No Live Cells", width / 2, height / 2);
        }
    }
}