const header = document.querySelector(".site-header");

function syncHeader() {
  header?.toggleAttribute("data-scrolled", window.scrollY > 12);
}

document.querySelectorAll("[data-copy-target]").forEach((button) => {
  button.addEventListener("click", async () => {
    const target = document.getElementById(button.dataset.copyTarget);
    const text = target?.innerText?.trim();
    if (!text) return;

    await navigator.clipboard.writeText(text);
    const previous = button.textContent;
    button.textContent = "Copied";
    window.setTimeout(() => {
      button.textContent = previous;
    }, 1300);
  });
});

syncHeader();
window.addEventListener("scroll", syncHeader, { passive: true });
