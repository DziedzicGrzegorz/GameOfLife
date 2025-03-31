// DOM elements
const input = document.getElementById("input");
const output = document.getElementById("output");
const formatBtn = document.getElementById("format-btn");

// Format function
async function format() {
    const jsonInput = input.value.trim();
    if (!jsonInput) {
        output.textContent = "Please enter some JSON.";
        return;
    }

    try {
        const pretty = await window.formatJSON(jsonInput);
        output.textContent = pretty;
    } catch (err) {
        output.textContent = `Error: ${err}`;
    }
}

// Event listeners
formatBtn.addEventListener("click", format);

// Optional: Format as you type (debounced)
let timeout;
input.addEventListener("input", () => {
    clearTimeout(timeout);
    timeout = setTimeout(format, 500); // Wait 500ms after typing stops
});