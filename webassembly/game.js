export async function initCanvases() {
    let c1 = startGameOfLife("game", 8, 1000);
    c1.setBackgroundColor("#111");
    c1.setColor("#ccc");
    c1.randomBirth(20);
    let currentInterval = 1000; // Initial interval in ms (matches startGameOfLife)
    const step = 50; // Adjust interval by 50ms per scroll
    const minInterval = 50; // Minimum: 50ms
    const maxInterval = 2000; // Maximum: 2000ms

    canvas.addEventListener("wheel", (e) => {
        e.preventDefault(); // Prevent page scrolling
        const delta = e.deltaY > 0 ? -step : step; // Scroll down decreases, up increases
        currentInterval = Math.min(maxInterval, Math.max(minInterval, currentInterval + delta));
        c1.setMinInterval(currentInterval);
        console.log(`Interval set to ${currentInterval}ms`); // Optional: for debugging
    });
    // Secret cheat code functionality
    let inputBuffer = "";
    document.addEventListener("keydown", (e) => {
        // Ignore modifier keys alone
        if (e.key === "Shift" || e.key === "Control" || e.key === "Alt") return;

        // Append typed character to buffer
        inputBuffer += e.key.toLowerCase();

        // Check for cheat codes
        if (inputBuffer.includes("color:red")) {
            c1.setBackgroundColor("red");
            inputBuffer = ""; // Reset buffer after match
        } else if (inputBuffer.includes("color:blue")) {
            c1.setBackgroundColor("blue");
            inputBuffer = "";
        } else if (inputBuffer.includes("color:green")) {
            c1.setBackgroundColor("green");
            inputBuffer = "";
        } else if (inputBuffer.includes("color:reset")) {
            c1.setBackgroundColor("#111"); // Reset to default
            inputBuffer = "";
        } else if (inputBuffer.includes("clear")) {
            c1.clear();
            inputBuffer = "";
        } else if (inputBuffer.includes("random")) {
            c1.randomBirth(50); // 50% random birth
            inputBuffer = "";
        } else if (inputBuffer.includes("stop")) {
            c1.stop();
            inputBuffer = "";
        } else if (inputBuffer.includes("resume")) {
            c1.resume();
            inputBuffer = "";
        }

        // Limit buffer size to prevent memory issues
        if (inputBuffer.length > 20) {
            inputBuffer = inputBuffer.slice(-10); // Keep last 10 characters
        }

        console.log(`Input buffer: ${inputBuffer}`); // Optional: for debugging
    })
}

let canvas = document.getElementById("game");

canvas.width = window.innerWidth;
canvas.height = window.innerHeight;

// window.addEventListener('resize', function(event){
//     canvas.width = window.innerWidth;
//     canvas.height = window.innerHeight;
//
//     let c1 = startGameOfLife("game", 8, 1000);
//     c1.setBackgroundColor("#111");
//     c1.setColor("#ccc");
//     c1.randomBirth(20);
// });

