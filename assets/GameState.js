export class GameState {
    constructor() {
        this.state = null;
    }

    update(data) {
        this.state = data;
        if (this.state.Board) {
            this.state.Board = this.state.Board.map(row => this.decodeBase64ToUint8Array(row));
        }
        console.log("[GameState] Updated state:", this.state);
    }

    decodeBase64ToUint8Array(base64) {
        const binaryString = atob(base64);
        const len = binaryString.length;
        const bytes = new Uint8Array(len);
        for (let i = 0; i < len; i++) {
            bytes[i] = binaryString.charCodeAt(i);
        }
        return bytes;
    }

    getState() {
        return this.state;
    }

    getLiveCells() {
        if (!this.state || !this.state.Board) return 0;
        let liveCells = 0;
        for (let y = 1; y < this.state.Height - 1; y++) {
            for (let x = 1; x < this.state.Width - 1; x++) {
                if (this.state.Board[y][x] >= 100) liveCells++;
            }
        }
        return liveCells;
    }
}